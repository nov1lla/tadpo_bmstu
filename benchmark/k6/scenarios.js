import http from 'k6/http';
import { check, sleep } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';

const state = {
  userID: null,
  gameID: null,
};

function randomName() {
  return `user_${Math.random().toString(36).slice(2, 10)}`;
}

function createUser() {
  const payload = JSON.stringify({ name: randomName() });
  const res = http.post(`${baseURL}/api/users`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'users.create', endpoint: 'users.create' },
  });
  check(res, { 'users.create status 201': (r) => r.status === 201 });
  return res.json();
}

function getUser(userID) {
  const res = http.get(`${baseURL}/api/users/${userID}`, {
    tags: { name: 'users.get', endpoint: 'users.get' },
  });
  check(res, { 'users.get status 200': (r) => r.status === 200 });
}

function createGame(userID) {
  const payload = JSON.stringify({
    userId: userID,
    playerColor: 'light',
    isPlayerFirst: true,
  });
  const res = http.post(`${baseURL}/api/games`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'games.create', endpoint: 'games.create' },
  });
  check(res, { 'games.create status 201': (r) => r.status === 201 });
  return res.json();
}

function getGameState(gameID) {
  const res = http.get(`${baseURL}/api/games/${gameID}`, {
    tags: { name: 'games.state', endpoint: 'games.state' },
  });
  check(res, { 'games.state status 200': (r) => r.status === 200 });
  return res.json();
}

function postUserMove(gameID, start, steps) {
  const payload = JSON.stringify({
    start,
    steps,
    requestOpponent: false,
  });
  const res = http.post(`${baseURL}/api/games/${gameID}/moves`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'games.move', endpoint: 'games.move' },
  });
  check(res, { 'games.move status 200': (r) => r.status === 200 });
  return res.status;
}

function postOpponentMove(gameID, start, steps) {
  const payload = JSON.stringify({
    start,
    steps,
  });
  const res = http.post(`${baseURL}/api/games/${gameID}/opponent-move-manual`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'games.opponent.move', endpoint: 'games.opponent.move' },
  });
  check(res, { 'games.opponent.move status 200': (r) => r.status === 200 });
  return res.status;
}

function key(row, col) {
  return `${row},${col}`;
}

function inside(row, col) {
  return row >= 0 && row < 8 && col >= 0 && col < 8;
}

function forwardDir(color) {
  return color === 'light' ? -1 : 1;
}

function buildIndex(board) {
  const posToPiece = {};
  board.pieces.forEach((p) => {
    posToPiece[key(p.row, p.col)] = { id: p.id, row: p.row, col: p.col, color: p.color, kind: p.kind };
  });
  return { posToPiece };
}

function listPieces(idx, color) {
  const pieces = [];
  Object.keys(idx.posToPiece).forEach((k) => {
    const p = idx.posToPiece[k];
    if (p && p.color === color) {
      pieces.push(p);
    }
  });
  pieces.sort((a, b) => (a.row - b.row) || (a.col - b.col) || a.id.localeCompare(b.id));
  return pieces;
}

function captureOptions(idx, piece, color) {
  const opts = [];
  const dirs = [
    { dr: 2, dc: 2 },
    { dr: 2, dc: -2 },
    { dr: -2, dc: 2 },
    { dr: -2, dc: -2 },
  ];
  for (let i = 0; i < dirs.length; i += 1) {
    const dir = dirs[i];
    if (piece.kind === 'man') {
      if (dir.dr / 2 !== forwardDir(color)) {
        continue;
      }
    }
    const landRow = piece.row + dir.dr;
    const landCol = piece.col + dir.dc;
    if (!inside(landRow, landCol)) {
      continue;
    }
    if (idx.posToPiece[key(landRow, landCol)]) {
      continue;
    }
    const midRow = piece.row + dir.dr / 2;
    const midCol = piece.col + dir.dc / 2;
    const victim = idx.posToPiece[key(midRow, midCol)];
    if (!victim || victim.color === color) {
      continue;
    }
    opts.push({ landRow, landCol, midRow, midCol });
  }
  opts.sort((a, b) => (a.landRow - b.landRow) || (a.landCol - b.landCol) || (a.midRow - b.midRow) || (a.midCol - b.midCol));
  return opts;
}

function stepOptions(idx, piece, color) {
  const opts = [];
  const dirs = [
    { dr: 1, dc: 1 },
    { dr: 1, dc: -1 },
    { dr: -1, dc: 1 },
    { dr: -1, dc: -1 },
  ];
  for (let i = 0; i < dirs.length; i += 1) {
    const dir = dirs[i];
    if (piece.kind === 'man') {
      if (dir.dr !== forwardDir(color)) {
        continue;
      }
    }
    const row = piece.row + dir.dr;
    const col = piece.col + dir.dc;
    if (!inside(row, col)) {
      continue;
    }
    if (idx.posToPiece[key(row, col)]) {
      continue;
    }
    opts.push({ row, col });
  }
  opts.sort((a, b) => (a.row - b.row) || (a.col - b.col));
  return opts;
}

function mustCapture(idx, color) {
  const pieces = listPieces(idx, color);
  for (let i = 0; i < pieces.length; i += 1) {
    if (captureOptions(idx, pieces[i], color).length > 0) {
      return true;
    }
  }
  return false;
}

function pickMove(idx, color) {
  const pieces = listPieces(idx, color);
  if (pieces.length === 0) {
    return null;
  }

  const needCapture = mustCapture(idx, color);

  if (needCapture) {
    for (let i = 0; i < pieces.length; i += 1) {
      const piece = pieces[i];
      const caps = captureOptions(idx, piece, color);
      if (caps.length === 0) {
        continue;
      }
      // Build full capture chain until no more captures for this piece.
      const steps = [];
      const current = { id: piece.id, row: piece.row, col: piece.col, color: piece.color, kind: piece.kind };
      // Clone occupancy map for simulation.
      const sim = { posToPiece: Object.assign({}, idx.posToPiece) };
      while (true) {
        const nextCaps = captureOptions(sim, current, color);
        if (nextCaps.length === 0) {
          break;
        }
        const cap = nextCaps[0];
        delete sim.posToPiece[key(current.row, current.col)];
        delete sim.posToPiece[key(cap.midRow, cap.midCol)];
        current.row = cap.landRow;
        current.col = cap.landCol;
        sim.posToPiece[key(current.row, current.col)] = current;
        steps.push({ row: cap.landRow, col: cap.landCol });
      }
      if (steps.length === 0) {
        continue;
      }
      return { start: { row: piece.row, col: piece.col }, steps };
    }
    return null;
  }

  for (let i = 0; i < pieces.length; i += 1) {
    const piece = pieces[i];
    const opts = stepOptions(idx, piece, color);
    if (opts.length === 0) {
      continue;
    }
    return { start: { row: piece.row, col: piece.col }, steps: [opts[0]] };
  }
  return null;
}

function determineTurnColor(game, movesCount) {
  const playerColor = game.playerColor;
  if (game.isPlayerFirst) {
    return movesCount % 2 === 0 ? playerColor : (playerColor === 'light' ? 'dark' : 'light');
  }
  return movesCount % 2 === 0 ? (playerColor === 'light' ? 'dark' : 'light') : playerColor;
}

function ensureUser() {
  if (!state.userID) {
    const user = createUser();
    state.userID = user.id;
  }
  return state.userID;
}

function ensureGame() {
  if (!state.gameID) {
    const gameResp = createGame(state.userID);
    state.gameID = gameResp.game.id;
  }
  return state.gameID;
}

function resetGame() {
  const gameResp = createGame(state.userID);
  state.gameID = gameResp.game.id;
  return state.gameID;
}


export const options = {
  systemTags: ['status', 'method', 'name', 'scenario'],
  scenarios: {
    degradation: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '5s', target: 10 },
        { duration: '5s', target: 30 },
        { duration: '10s', target: 50 },
      ],
      tags: { phase: 'degradation' },
    },
    peak: {
      executor: 'constant-vus',
      vus: 50,
      duration: '25s',
      startTime: '20s',
      tags: { phase: 'peak' },
    },
    recovery: {
      executor: 'ramping-vus',
      startVUs: 50,
      stages: [
        { duration: '10s', target: 10 },
        { duration: '10s', target: 10 },
        { duration: '30s', target: 10 },

      ],
      startTime: '45s',
      tags: { phase: 'recovery' },
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<2000'],
  },
};

export default function () {
  const userID = ensureUser();
  ensureGame();

  const stateResp = getGameState(state.gameID);
  const game = stateResp.game;
  const movesCount = (stateResp.moves || []).length;
  const turnColor = determineTurnColor(game, movesCount);
  const idx = buildIndex(stateResp.board);
  const move = pickMove(idx, turnColor);
  if (!move) {
    resetGame();
    sleep(0.1);
    return;
  }

  let status = 0;
  if (turnColor === game.playerColor) {
    status = postUserMove(state.gameID, move.start, move.steps);
  } else {
    status = postOpponentMove(state.gameID, move.start, move.steps);
  }
  getGameState(state.gameID);

  if (status !== 200) {
    resetGame();
  } else if (Math.random() < 0.05) {
    getUser(userID);
  }

  sleep(0.1);
}
