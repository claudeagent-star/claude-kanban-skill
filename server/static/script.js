// Kanban frontend, vanilla JS, no deps. Fetches state from /api/cards,
// renders 5 columns, supports drag-and-drop between AND within columns
// (with an insertion indicator), and a modal for create / edit / delete.

const COLUMNS = [
  { id: 'to-do',       label: 'To Do' },
  { id: 'blocked',     label: 'Blocked' },
  { id: 'in-progress', label: 'In Progress' },
  { id: 'in-review',   label: 'In Review' },
  { id: 'done',        label: 'Done' },
];

const boardEl = document.querySelector('.board');
const modal     = document.getElementById('card-modal');
const form      = document.getElementById('card-form');
const titleEl   = document.getElementById('card-title');
const descEl    = document.getElementById('card-description');
const colEl     = document.getElementById('card-column');
const colorEl   = document.getElementById('card-color');
const delBtn    = document.getElementById('card-delete');
const cancelBtn = document.getElementById('card-cancel');
const addBtn    = document.getElementById('add-card');

let editingId = null; // null while creating, card.id while editing

// ===== API =====

async function apiList() {
  const res = await fetch('api/cards');
  if (!res.ok) throw new Error('list failed');
  return res.json();
}
async function apiCreate(card) {
  const res = await fetch('api/cards', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(card),
  });
  if (!res.ok) throw new Error('create failed: ' + res.status);
  return res.json();
}
async function apiUpdate(id, patch) {
  const res = await fetch('api/cards/' + encodeURIComponent(id), {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(patch),
  });
  if (!res.ok) throw new Error('update failed: ' + res.status);
  return res.json();
}
async function apiDelete(id) {
  const res = await fetch('api/cards/' + encodeURIComponent(id), { method: 'DELETE' });
  if (!res.ok) throw new Error('delete failed: ' + res.status);
}

// ===== Drag-and-drop helpers =====

// Single shared indicator line that marks the drop slot.
let dropIndicator = null;
function getDropIndicator() {
  if (!dropIndicator) {
    dropIndicator = document.createElement('div');
    dropIndicator.className = 'drop-indicator';
  }
  return dropIndicator;
}
function clearDropIndicator() {
  if (dropIndicator && dropIndicator.parentNode) {
    dropIndicator.parentNode.removeChild(dropIndicator);
  }
}

// Compute the index, within a column body, where a drop at clientY would land.
// Excludes the card being dragged so reordering within a column behaves as expected.
function dropIndexAt(body, clientY) {
  const cards = [...body.querySelectorAll('.card:not(.dragging)')];
  for (let i = 0; i < cards.length; i++) {
    const r = cards[i].getBoundingClientRect();
    if (clientY < r.top + r.height / 2) return i;
  }
  return cards.length;
}

function placeDropIndicator(body, index) {
  const cards = [...body.querySelectorAll('.card:not(.dragging)')];
  const ind = getDropIndicator();
  if (index >= cards.length) body.appendChild(ind);
  else body.insertBefore(ind, cards[index]);
}

// ===== Rendering =====

function render(cards) {
  boardEl.innerHTML = '';
  const byCol = Object.fromEntries(COLUMNS.map(c => [c.id, []]));
  for (const card of cards) {
    if (!byCol[card.column]) byCol[card.column] = [];
    byCol[card.column].push(card);
  }
  for (const col of COLUMNS) {
    const colCards = (byCol[col.id] || []).sort((a, b) => a.position - b.position);
    boardEl.appendChild(renderColumn(col, colCards));
  }
}

function renderColumn(col, cards) {
  const colEl = document.createElement('section');
  colEl.className = 'column';
  colEl.dataset.id = col.id;
  colEl.innerHTML = `
    <header class="column-header">
      <span>${col.label}</span>
      <span class="count">${cards.length}</span>
    </header>
    <div class="column-body"></div>
  `;
  const body = colEl.querySelector('.column-body');

  // Drop target on the column body. Computes insertion index from cursor Y,
  // shows an indicator line at that slot, and on drop PATCHes column+position.
  body.addEventListener('dragover', e => {
    e.preventDefault();
    if (e.dataTransfer) e.dataTransfer.dropEffect = 'move';
    body.classList.add('drag-over');
    placeDropIndicator(body, dropIndexAt(body, e.clientY));
  });
  body.addEventListener('dragleave', e => {
    // Only clear when leaving the body for somewhere outside its subtree.
    if (e.relatedTarget && body.contains(e.relatedTarget)) return;
    body.classList.remove('drag-over');
    clearDropIndicator();
  });
  body.addEventListener('drop', async e => {
    e.preventDefault();
    body.classList.remove('drag-over');
    const cardId = e.dataTransfer.getData('text/card-id');
    const index = dropIndexAt(body, e.clientY);
    clearDropIndicator();
    if (!cardId) return;
    try {
      await apiUpdate(cardId, { column: col.id, position: index });
      reload();
    } catch (err) {
      console.error(err);
      alert('Move failed: ' + err.message);
    }
  });

  for (const card of cards) {
    body.appendChild(renderCard(card));
  }
  return colEl;
}

function renderCard(card) {
  const el = document.createElement('article');
  el.className = 'card';
  el.draggable = true;
  el.dataset.id = card.id;
  if (card.color) {
    el.dataset.color = card.color;
  }

  const title = document.createElement('div');
  title.className = 'title';
  title.textContent = card.title;
  el.appendChild(title);

  if (card.description) {
    const desc = document.createElement('div');
    desc.className = 'desc-preview';
    desc.textContent = card.description;
    el.appendChild(desc);
  }

  el.addEventListener('dragstart', e => {
    e.dataTransfer.setData('text/card-id', card.id);
    e.dataTransfer.effectAllowed = 'move';
    el.classList.add('dragging');
  });
  el.addEventListener('dragend', () => {
    el.classList.remove('dragging');
    clearDropIndicator();
  });

  el.addEventListener('click', () => openModal(card));

  return el;
}

// ===== Modal =====

function openModal(card) {
  if (card) {
    editingId = card.id;
    titleEl.value = card.title;
    descEl.value = card.description || '';
    colEl.value = card.column || 'to-do';
    colorEl.value = card.color || '';
    delBtn.hidden = false;
  } else {
    editingId = null;
    titleEl.value = '';
    descEl.value = '';
    colEl.value = 'to-do';
    colorEl.value = '';
    delBtn.hidden = true;
  }
  modal.showModal();
  setTimeout(() => titleEl.focus(), 0);
}

cancelBtn.addEventListener('click', () => modal.close());

form.addEventListener('submit', async e => {
  e.preventDefault();
  const payload = {
    title: titleEl.value.trim(),
    description: descEl.value,
    column: colEl.value,
    color: colorEl.value,
  };
  if (!payload.title) return;
  try {
    if (editingId) {
      await apiUpdate(editingId, payload);
    } else {
      await apiCreate(payload);
    }
    modal.close();
    reload();
  } catch (err) {
    console.error(err);
    alert('Save failed: ' + err.message);
  }
});

delBtn.addEventListener('click', async () => {
  if (!editingId) return;
  if (!confirm('Delete this card?')) return;
  try {
    await apiDelete(editingId);
    modal.close();
    reload();
  } catch (err) {
    console.error(err);
    alert('Delete failed: ' + err.message);
  }
});

addBtn.addEventListener('click', () => openModal(null));

// ===== Boot =====

async function reload() {
  try {
    render(await apiList());
  } catch (err) {
    console.error(err);
    boardEl.innerHTML = '<p style="padding:1rem;color:#b54848">Failed to load: ' + err.message + '</p>';
  }
}

reload();
