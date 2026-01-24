import http from 'k6/http';
import { check, sleep } from 'k6';

const baseURL = __ENV.BASE_URL || 'http://localhost:8080';

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

function postMove(gameID) {
  const payload = JSON.stringify({
    start: { row: 5, col: 0 },
    steps: [{ row: 4, col: 1 }],
    requestOpponent: false,
  });
  const res = http.post(`${baseURL}/api/games/${gameID}/moves`, payload, {
    headers: { 'Content-Type': 'application/json' },
    tags: { name: 'games.move', endpoint: 'games.move' },
  });
  check(res, { 'games.move status 200': (r) => r.status === 200 });
}

function getMoves(gameID) {
  const res = http.get(`${baseURL}/api/games/${gameID}/moves`, {
    tags: { name: 'games.moves', endpoint: 'games.moves' },
  });
  check(res, { 'games.moves status 200': (r) => r.status === 200 });
}

function deleteGame(gameID) {
  const res = http.del(`${baseURL}/api/games/${gameID}`, null, {
    tags: { name: 'games.delete', endpoint: 'games.delete' },
  });
  check(res, { 'games.delete status 204': (r) => r.status === 204 });
}

// Нагрузка и сценарии:
// - degradation: плавный рост нагрузки
// - peak: удержание максимальной нагрузки
// - recovery: спад после перегруза
// Сохраняем метрики latency и перцентили по всем запросам и по каждому endpoint (через tags).
export const options = {
  systemTags: ['status', 'method', 'name', 'scenario'],
  scenarios: {
    degradation: {
      executor: 'ramping-vus',
      startVUs: 1,
      stages: [
        { duration: '30s', target: 10 },
        { duration: '30s', target: 30 },
        { duration: '30s', target: 50 },
      ],
      tags: { phase: 'degradation' },
    },
    peak: {
      executor: 'constant-vus',
      vus: 50,
      duration: '30s',
      startTime: '90s',
      tags: { phase: 'peak' },
    },
    recovery: {
      executor: 'ramping-vus',
      startVUs: 50,
      stages: [
        { duration: '20s', target: 20 },
        { duration: '20s', target: 5 },
      ],
      startTime: '120s',
      tags: { phase: 'recovery' },
    },
  },
  thresholds: {
    http_req_failed: ['rate<0.01'],
    http_req_duration: ['p(95)<2000'],
  },
};

export default function () {
  const user = createUser();
  const userID = user.id;
  getUser(userID);

  const gameResp = createGame(userID);
  const gameID = gameResp.game.id;
  postMove(gameID);
  getMoves(gameID);
  deleteGame(gameID);

  sleep(0.1);
}
