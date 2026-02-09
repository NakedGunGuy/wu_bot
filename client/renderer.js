const GameClient = require("../modules/GameClient.js");

const CORPORATIONS_TYPES = {
  NONE: 0,
  U: 1,
  E: 2,
  R: 3,
}

const OBJECT_TYPES = {
  NORMAL_TELEPORT: "NORMAL_TELEPORT",
  SPACE_STATION: "SPACE_STATION",
  TRADE_STATION: "TRADE_STATION",
  QUEST_STATION: "QUEST_STATION",
}

const COLLECTABLE_TYPES = {
  BONUS_BOX: 0,
  CARGO_BOX: 1,
  GREEN_BOX: 3,
}

const NPC_TYPES = [
  "-=(Hydro)=-",
  "-=(Hyper|Hydro)=-",
  "-=(Jenta)=-",
  "-=(Hyper|Jenta)=-",
  "-=(Mali)=-",
  "-=(Hyper|Mali)=-",
  "-=(Plarion)=-",
  "-=(Hyper|Plarion)=-",
  "-=(Motron)=-",
  "-=(Hyper|Motron)=-",
  "-=(Xeon)=-",
  "-=(Hyper|Xeon)=-",
  "-=(Bangoliour)=-",
  "-=(Hyper|Bangoliour)=-",
  "-=(Zavientos)=-",
  "-=(Magmius)=-",
  "-=(Hyper|Magmius)=-",
  "-=(Raider)=-",
  "-=(Hyper|Raider)=-",
  "-=(Vortex)=-",
  "-=(Hyper|Vortex)=-",
  "-=(Quattroid)=-",
];

const COLLECTABLE_DATA_TABLE = [
  { type: COLLECTABLE_TYPES.BONUS_BOX, name: "Bonus box", storageKey: `collectable-${COLLECTABLE_TYPES.BONUS_BOX}` },
  { type: COLLECTABLE_TYPES.CARGO_BOX, name: "Cargo box", storageKey: `collectable-${COLLECTABLE_TYPES.CARGO_BOX}` },
  { type: COLLECTABLE_TYPES.GREEN_BOX, name: "Green box", storageKey: `collectable-${COLLECTABLE_TYPES.GREEN_BOX}` },
];

const KILLABLE_DATA_TABLE = NPC_TYPES.map(name => ({
  name,
  storageKey: `killable-${name}`
}));

let client;

document.addEventListener("DOMContentLoaded", () => {
  const adapterSettings = () => {
    return {
      workMap: JSON.parse(localStorage.getItem("workMap")) || "U-1",
      killTargets: [
        ...Object.keys(localStorage)
          .filter(key => key.startsWith("target>killable-"))
          .map(key => {
            const name = key.replace("target>killable-", "");
            return {
              name,
              priority: JSON.parse(localStorage.getItem(`priority>killable-${name}`)) || 1,
              ammo: JSON.parse(localStorage.getItem(`ammo>killable-${name}`)) || 1,
              rockets: JSON.parse(localStorage.getItem(`rocket>killable-${name}`)) || 1,
              farmNearPortal: false
            };
          }),
      ],
      collectBoxTypes: [
        ...Object.keys(localStorage)
          .filter(key => key.startsWith("target>collectable-"))
          .map(key => {
            const type = parseInt(key.replace("target>collectable-", ""), 10);
            return {
              type,
              priority: localStorage.getItem(`priority>collectable-${type}`) ? Number(JSON.parse(localStorage.getItem(`priority>collectable-${type}`))) : 1
            };
          }),
      ],
      minHP: JSON.parse(localStorage.getItem("minHP")) || 10,
      adviceHP: JSON.parse(localStorage.getItem("adviceHP")) || 70,
      kill: {
        targetEngagedNPC: JSON.parse(localStorage.getItem("targetEngagedNPC")) || false,
      },
      admin: {
        enabled: JSON.parse(localStorage.getItem("detectAdmin")) || false,
        delay: JSON.parse(localStorage.getItem("adminDelay")) || 5,
      },
      escape: {
        enabled: JSON.parse(localStorage.getItem("escape")) || false,
        delay: JSON.parse(localStorage.getItem("escapeDelay")) || 20000,
      },
      config: {
        switchConfigOnShieldsDown: JSON.parse(localStorage.getItem("switchConfigOnShieldsDown")) || false,
        attacking: JSON.parse(localStorage.getItem("attacking")) || 1,
        fleeing: JSON.parse(localStorage.getItem("fleeing")) || 2,
        flying: JSON.parse(localStorage.getItem("flying")) || 2,
      },
      enrichment: {
        lasers: { enabled: false, materialType: 0 },
        rockets: { enabled: false, materialType: 0 },
        shields: { enabled: false, materialType: 0 },
        speed: { enabled: false, materialType: 0 },
      },
      autobuy: {
        laser: {
          RLX_1: JSON.parse(localStorage.getItem("autoBuy>RLX_1")) || false,
          GLX_2: JSON.parse(localStorage.getItem("autoBuy>GLX_2")) || false,
          BLX_3: JSON.parse(localStorage.getItem("autoBuy>BLX_3")) || false,
          GLX_2_AS: JSON.parse(localStorage.getItem("autoBuy>GLX_2_AS")) || false,
          MRS_6X: false,
        },
        rockets: {
          KEP_410: JSON.parse(localStorage.getItem("autoBuy>KEP_410")) || false,
          NC_30: JSON.parse(localStorage.getItem("autoBuy>NC_30")) || false,
          TNC_130: JSON.parse(localStorage.getItem("autoBuy>TNC_130")) || false,
        },
        key: {
          enabled: false,
          savePLT: 50000,
        },
      },
      break: {
        interval: 0,
        duration: 0,
      }
    }
  }

  // Initialize the tabs
  document.querySelectorAll("#tabs > div").forEach((tab, index) => {
    if (index === 0) {
      tab.classList.add("bg-zinc-950");
      document.querySelector(`#${tab.id.split("-")[1]}`).classList.remove("hidden");
    }
    tab.addEventListener("click", () => {
      document.querySelectorAll("#content > div").forEach(content => {
        content.classList.add("hidden");
      });
      document.querySelector(`#${tab.id.split("-")[1]}`).classList.remove("hidden");

      document.querySelectorAll("#tabs > div").forEach(tab => {
        tab.classList.remove("bg-zinc-950");
      });
      tab.classList.add("bg-zinc-950");
    });
  });

  // Initialize the NPC table
  const npcList = document.querySelector("tbody#npcList");

  KILLABLE_DATA_TABLE.forEach(npc => {
    const row = document.createElement("tr");
    row.innerHTML = `<td class="mb-2">
              <input
                type="checkbox"
                data-storage-key="target>${npc.storageKey}"
              />
            </td>

            <td class="px-2">
              <input
                type="text"
                class="w-full disabled:bg-gray-600/50"
                disabled
                data-storage-key="name>${npc.storageKey}"
                value="${npc.name}"
              />
            </td>

            <td class="px-2">
              <input
                type="number"
                data-storage-key="priority>${npc.storageKey}"
                placeholder="Priority"
                min="1"
                max="999"
              />
            </td>

            <td class="px-2">
              <select
                data-storage-key="ammo>${npc.storageKey}"
              >
                <option value="1" ${npc.ammo === 1 ? "selected" : ""}>RLX_1</option>
                <option value="2" ${npc.ammo === 2 ? "selected" : ""}>GLX_2</option>
                <option value="3" ${npc.ammo === 3 ? "selected" : ""}>BLX_3</option>
                <option value="4" ${npc.ammo === 4 ? "selected" : ""}>GLX_2_AS</option>
                <option value="5" ${npc.ammo === 5 ? "selected" : ""}>MRS_6X</option>
              </select>
            </td>

            <td class="px-2">
              <select
                data-storage-key="rocket>${npc.storageKey}"
              >
                <option value="1" ${npc.rockets === 1 ? "selected" : ""}>KEP_410</option>
                <option value="2" ${npc.rockets === 2 ? "selected" : ""}>NC_30</option>
                <option value="3" ${npc.rockets === 3 ? "selected" : ""}>TNC_130</option>
              </select>
            </td>`;
    npcList.appendChild(row);
  });

  // Initialize the collectable table
  const collectableList = document.querySelector("tbody#collectableList");

  COLLECTABLE_DATA_TABLE.forEach(collectable => {
    const row = document.createElement("tr");
    row.innerHTML = `
            <td class="mb-2">
              <input
                type="checkbox"
                data-storage-key="target>${collectable.storageKey}"
              />
            </td>

            <td class="px-2">
              <input
                type="text"
                class="w-full disabled:bg-gray-600/50"
                value="${collectable.name}"
                disabled
              />
            </td>

            <td class="px-2">
              <input
                type="number"
                data-storage-key="priority>${collectable.storageKey}"
                placeholder="Priority"
                min="1"
                max="999"
              />
            </td>
  `;
    collectableList.appendChild(row);
  });

  document.querySelectorAll('input[data-storage-key], select[data-storage-key]').forEach(input => {
    input.addEventListener('change', (event) => {
      const key = event.target.getAttribute('data-storage-key');
      const value = event.target.type === 'checkbox' ? event.target.checked : event.target.value;
      saveToStorage(key, value);
      client.setMode(JSON.parse(localStorage.getItem("mode")) || "collect");
      client.setSettings(adapterSettings());
    });
  });

  function loadFromStorage() {
    const elements = document.querySelectorAll('[data-storage-key]');
    elements.forEach(element => {
      const key = element.getAttribute('data-storage-key');
      const savedValue = localStorage.getItem(key);

      if (savedValue !== null) {
        const parsedValue = JSON.parse(savedValue);

        if (element.type === 'checkbox') {
          element.checked = parsedValue;
        } else if (element.tagName === 'SELECT' || element.tagName === 'INPUT') {
          element.value = parsedValue;
        }
      }
    });
  }

  loadFromStorage();

  function saveToStorage(key, value) {
    localStorage.setItem(key, JSON.stringify(value));
  }

  document.querySelector("#play").addEventListener("click", () => {
    client.setSettings(adapterSettings());
    client.setMode(JSON.parse(localStorage.getItem("mode")) || "collect");
    client.start();
  })

  document.querySelector("#pause").addEventListener("click", () => {
    client.stop();
  })

  // Game rendering
  const canvas = document.querySelector("canvas");
  const ctx = canvas.getContext("2d");
  canvas.width = 800;
  canvas.height = 600;
  canvas.classList.add("overflow-hidden");

  let currentMapWidth = 0;
  let currentMapHeight = 0;

  let playerId = null;

  let scaleX = 1
  let scaleY = 1;

  const scalePosition = ({ x, y }) => ({
    x: Math.round(x * scaleX),
    y: Math.round(y * scaleY),
  });

  const drawEnemy = ({ color = "red", width = 5, height = 5, x, y }) => {
    const pos = scalePosition({ x, y });
    ctx.fillStyle = color;
    ctx.fillRect(pos.x, pos.y, width, height);
  }

  const drawPortal = ({ color = "gray", width = 15, height = 15, x, y }) => {
    const pos = scalePosition({ x, y });
    ctx.strokeStyle = color;
    ctx.lineWidth = 2;
    ctx.beginPath();
    ctx.arc(pos.x, pos.y, Math.min(width, height) / 2, 0, Math.PI * 2);
    ctx.closePath();
    ctx.stroke();
  }

  const drawPlayer = ({ color = "white", width = 8, height = 8, x, y, cross = false }) => {
    const pos = scalePosition({ x, y });
    ctx.fillStyle = color;
    ctx.fillRect(pos.x, pos.y, width, height);
    ctx.strokeStyle = color;
    ctx.lineWidth = 1;

    if (!cross) return;

    const crossColor = `rgba(255, 255, 255, 0.5)`;

    ctx.beginPath();
    ctx.strokeStyle = crossColor
    ctx.moveTo(0, pos.y + height / 2);
    ctx.lineTo(canvas.width, pos.y + height / 2);
    ctx.stroke();

    ctx.beginPath();
    ctx.strokeStyle = crossColor
    ctx.moveTo(pos.x + width / 2, 0);
    ctx.lineTo(pos.x + width / 2, canvas.height);
    ctx.stroke();
  };

  const drawText = ({ text, x, y, size = 11, opacity = 1, color }) => {
    color ? ctx.fillStyle = color : ctx.fillStyle = `rgba(255, 255, 255, ${opacity})`;
    ctx.font = `${size}px Arial`;
    ctx.fillText(text, x, y);
  }

  const drawSpaceStation = ({ x, y, radius = 15 }) => {
    const color = `rgba(255, 255, 255, 0.5)`;
    const pos = scalePosition({ x, y });
    ctx.fillStyle = color
    ctx.beginPath();
    ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2);
    ctx.closePath();
    ctx.fill();

    drawSurroundingCircles({ x: pos.x, y: pos.y, radius, smallRadius: radius / 3, count: 12, color });
  }

  const drawQuestStation = ({ x, y, radius = 10 }) => {
    const pos = scalePosition({ x, y });
    ctx.fillStyle = `rgba(103, 27, 255, 0.79)`;
    ctx.beginPath();
    ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2);
    ctx.closePath();
    ctx.fill();
  }

  const drawSurroundingCircles = ({
                                    x,
                                    y,
                                    radius,
                                    smallRadius = radius / 3,
                                    count = 12,
                                    color = `rgba(255, 255, 255, 0.9)`
                                  }) => {
    const angleStep = (2 * Math.PI) / count;
    ctx.fillStyle = color;
    for (let i = 0; i < count; i++) {
      const angle = i * angleStep;
      const smallX = x + (radius + smallRadius) * Math.cos(angle);
      const smallY = y + (radius + smallRadius) * Math.sin(angle);
      ctx.beginPath();
      ctx.arc(smallX, smallY, smallRadius, 0, Math.PI * 2);
      ctx.closePath();
      ctx.fill();
    }
  };

  const drawTradeStation = ({ x, y, radius = 10 }) => {
    const pos = scalePosition({ x, y });
    ctx.fillStyle = `rgba(188, 143, 61, 0.94)`;
    ctx.beginPath();
    ctx.arc(pos.x, pos.y, radius, 0, Math.PI * 2);
    ctx.closePath();
    ctx.fill();
  };

  const drawBar = ({ x, y, percentage = 100, height, color = "white", backgroundColor = "gray" }) => {
    const totalWidth = 340;
    const filledWidth = (percentage / 100) * totalWidth;

    ctx.fillStyle = backgroundColor;
    ctx.fillRect(x, y, totalWidth, height);

    ctx.fillStyle = color;
    ctx.fillRect(x, y, filledWidth, height);
  };

  const drawLine = ({ x1, y1, x2, y2, color = "white", width = 1 }) => {
    ctx.strokeStyle = color;
    ctx.lineWidth = width;
    ctx.beginPath();
    ctx.moveTo(x1, y1);
    ctx.lineTo(x2, y2);
    ctx.stroke();
  }

  const drawDashedCircle = ({ x, y, radius, color = "white", lineWidth = 1, dash = [4, 4] }) => {
    const pos = scalePosition({ x, y });
    const scaledRadius = radius * ((scaleX + scaleY) / 2);
    ctx.strokeStyle = color;
    ctx.lineWidth = lineWidth;
    ctx.setLineDash(dash);
    ctx.beginPath();
    ctx.arc(pos.x, pos.y, scaledRadius, 0, Math.PI * 2);
    ctx.stroke();
    ctx.setLineDash([]);
  }

  const drawCircle = ({ x, y, radius, color = "white", lineWidth = 1 }) => {
    const pos = scalePosition({ x, y });
    const scaledRadius = radius * ((scaleX + scaleY) / 2);
    ctx.strokeStyle = color;
    ctx.lineWidth = lineWidth;
    ctx.beginPath();
    ctx.arc(pos.x, pos.y, scaledRadius, 0, Math.PI * 2);
    ctx.stroke();
  }

  const drawCollectable = ({ x, y, type, width = 3, height = 3 }) => {
    let color;

    switch (type) {
      case COLLECTABLE_TYPES.BONUS_BOX:
        color = "#ffdb00";
        break;
      case COLLECTABLE_TYPES.CARGO_BOX:
        color = "#c58a59";
        break;
      case COLLECTABLE_TYPES.GREEN_BOX:
        color = "#a4ffb4";
        break;
      default:
        color = "white";
    }

    const pos = scalePosition({ x, y });
    ctx.fillStyle = color;
    ctx.fillRect(pos.x, pos.y, width, height);
  }

  client = new GameClient({
    username: JSON.parse(localStorage.getItem("username")) || "",
    password: JSON.parse(localStorage.getItem("password")) || "",
    serverId: JSON.parse(localStorage.getItem("server")) || "eu1",
  });

  // Event log system
  const logEntries = [];
  const MAX_LOG_ENTRIES = 200;
  const LOG_COLORS = {
    kill: "#ff4444",
    collect: "#ffdb00",
    health: "#44ff44",
    escape: "#ff8800",
    default: "#cccccc",
  };

  const originalConsoleLog = console.log;
  console.log = function (...args) {
    originalConsoleLog.apply(console, args);
    const message = args.map(a => typeof a === "object" ? JSON.stringify(a) : String(a)).join(" ");
    let type = "default";
    const msgLower = message.toLowerCase();
    if (msgLower.includes("kill") || msgLower.includes("attack") || msgLower.includes("npc")) type = "kill";
    else if (msgLower.includes("collect") || msgLower.includes("box")) type = "collect";
    else if (msgLower.includes("health") || msgLower.includes("recover") || msgLower.includes("shield")) type = "health";
    else if (msgLower.includes("escape") || msgLower.includes("enemy") || msgLower.includes("admin")) type = "escape";

    logEntries.push({ time: new Date(), message, type });
    if (logEntries.length > MAX_LOG_ENTRIES) logEntries.shift();

    // Update log panel if visible
    const logList = document.querySelector("#log-list");
    if (logList) {
      const entry = document.createElement("div");
      entry.style.color = LOG_COLORS[type] || LOG_COLORS.default;
      entry.style.fontSize = "11px";
      entry.style.fontFamily = "monospace";
      entry.style.padding = "1px 4px";
      const timeStr = new Date().toLocaleTimeString();
      entry.textContent = `[${timeStr}] ${message}`;
      logList.appendChild(entry);
      if (logList.children.length > MAX_LOG_ENTRIES) logList.removeChild(logList.firstChild);
      logList.scrollTop = logList.scrollHeight;
    }
  };

  canvas.addEventListener("click", (event) => {
    const rect = canvas.getBoundingClientRect();
    const x = ((event.clientX - rect.left) / canvas.width) * currentMapWidth;
    const y = ((event.clientY - rect.top) / canvas.height) * currentMapHeight;
    client.scene.move(x, y);
  });

  const render = () => {
    if (!client.client.clientLoaded) return requestAnimationFrame(render);
    if (!client.scene.playerId) return requestAnimationFrame(render);
    playerId = client.scene.playerId
    ctx.clearRect(0, 0, canvas.width, canvas.height);

    currentMapWidth = client.scene.currentMapWidth || currentMapWidth;
    currentMapHeight = client.scene.currentMapHeight || currentMapHeight;

    scaleX = canvas.width / currentMapWidth;
    scaleY = canvas.height / currentMapHeight;

    drawText({
      text: client.stats.messageState,
      x: canvas.width / 2 - ctx.measureText(client.stats.messageState).width / 2,
      y: 10
    });

    // Status top left
    drawText({ text: `Status: ${client.status}`, x: 10, y: 20 });
    drawText({ text: `Server: ${client.client.serverId.toUpperCase()}`, x: 10, y: 35 });

    // Name of the map (Middle - screen)
    drawText({ text: client.scene.currentMap, x: canvas.width / 2 - 50, y: canvas.height / 2, size: 50, opacity: 0.2 });

    // Static elements
    const player = client.scene.ships[playerId];
    const selected = player?.selected;

    drawBar({
      x: 10,
      y: canvas.height - 35,
      height: 12,
      color: "green",
      percentage: (player?.health / player?.maxHealth) * 100
    });
    drawBar({
      x: 10, y: canvas.height - 20, height: 12, color: "blue",
      percentage: (player?.shield / player?.maxShield) * 100
    });

    drawText({ text: `${player?.health} / ${player?.maxHealth}`, x: 20, y: canvas.height - 25 });
    drawText({ text: `${player?.shield} / ${player?.maxShield}`, x: 20, y: canvas.height - 10 });

    if (selected) {
      const npc = client.scene.ships[selected];
      if (npc) {
        const barX = 435;
        const barY = canvas.height - 35;
        const textX = 445;

        drawBar({ x: barX, y: barY, height: 12, color: "green", percentage: (npc.health / npc.maxHealth) * 100 });
        drawBar({ x: barX, y: barY + 15, height: 12, color: "blue", percentage: (npc.shield / npc.maxShield) * 100 });

        drawText({ text: `${npc.health} / ${npc.maxHealth}`, x: textX, y: barY + 10 });
        drawText({ text: `${npc.shield} / ${npc.maxShield}`, x: textX, y: barY + 25 });

        drawText({
          text: npc.name,
          x: 580 - ctx.measureText(npc.name).width / 2,
          y: canvas.height - 45,
          color: "red",
          size: 18,
        });
      }
    }

    // Environment elements
    Object.values(client.scene.currentMapObjects).forEach(pos => {
      pos.type === OBJECT_TYPES.NORMAL_TELEPORT && drawPortal({
        x: pos.x,
        y: pos.y,
      })

      pos.type === OBJECT_TYPES.SPACE_STATION && drawSpaceStation({
        x: pos.x,
        y: pos.y,
      })

      pos.type === OBJECT_TYPES.TRADE_STATION && drawTradeStation({
        x: pos.x,
        y: pos.y,
      })

      pos.type === OBJECT_TYPES.QUEST_STATION && drawQuestStation({
        x: pos.x,
        y: pos.y,
      })
    });

    // Collectables
    Object.values(client.scene.collectibles).forEach(pos => drawCollectable({
      x: pos.x,
      y: pos.y,
      type: pos.type
    }));

    // Detection radius circle (2000 units) around player
    if (player) {
      drawDashedCircle({
        x: player.x, y: player.y,
        radius: 2000,
        color: "rgba(255, 255, 255, 0.15)",
        lineWidth: 1,
        dash: [6, 6]
      });
    }

    // Orbit radius circle when orbiting
    const stateManager = client.controller?.StateManager;
    if (stateManager?.navigation?.orbiting && player && selected) {
      const targetShip = client.scene.ships[selected];
      if (targetShip) {
        drawDashedCircle({
          x: targetShip.x, y: targetShip.y,
          radius: 400,
          color: "rgba(0, 200, 255, 0.4)",
          lineWidth: 1,
          dash: [4, 4]
        });
      }
    }

    // Ships
    const ownCorporation = Object.values(client.scene.ships).find((ship) => ship.id === playerId)?.corporation;
    Object.values(client.scene.ships).forEach((ship) => {
      const pos = scalePosition({ x: ship.x, y: ship.y });

      if (ship.id === playerId) {
        drawPlayer({ x: ship.x, y: ship.y, cross: true });
        drawText({
          x: 120,
          y: 555,
          text: ship.name,
          color: "white",
          size: 18
        })
      } else if (ship.corporation === ownCorporation) {
        drawPlayer({ x: ship.x, y: ship.y, color: "blue" });
        drawText({
          text: ship.name,
          x: pos.x - ctx.measureText(ship.name).width / 2,
          y: pos.y - 10,
          color: "blue",
          size: 10
        });
      } else if (ship.corporation === CORPORATIONS_TYPES.NONE) {
        drawEnemy({ x: ship.x, y: ship.y });
        drawText({
          text: ship.name,
          x: pos.x - ctx.measureText(ship.name).width / 2,
          y: pos.y - 10,
          color: "red",
          size: 8
        });
      } else {
        drawPlayer({ x: ship.x, y: ship.y, color: "orange", width: 10, height: 10 });
        drawText({
          text: ship.name,
          x: pos.x - ctx.measureText(ship.name).width / 2,
          y: pos.y - 10,
          color: "orange",
          size: 10
        });
      }

      // Direction indicator: draw a small line from ship toward its target
      if (ship.isMoving && ship.targetX != null && ship.targetY != null) {
        const dx = ship.targetX - ship.x;
        const dy = ship.targetY - ship.y;
        const dist = Math.sqrt(dx * dx + dy * dy);
        if (dist > 0) {
          const dirLen = 15; // pixels on screen
          const nx = dx / dist;
          const ny = dy / dist;
          drawLine({
            x1: pos.x + 3, y1: pos.y + 3,
            x2: pos.x + 3 + nx * dirLen, y2: pos.y + 3 + ny * dirLen,
            color: ship.id === playerId ? "rgba(255,255,255,0.5)" : "rgba(255,100,100,0.4)",
            width: 1
          });
        }
      }

      // Attack lines: orange lines from enemy ships attacking the player
      if (ship.id !== playerId && ship.selected === playerId && ship.isAttacking) {
        const playerPos = scalePosition({ x: player.x, y: player.y });
        drawLine({
          x1: pos.x + 3, y1: pos.y + 3,
          x2: playerPos.x + 4, y2: playerPos.y + 4,
          color: "rgba(255, 140, 0, 0.6)",
          width: 1
        });
      }
    });

    // Red line from player to selected NPC target
    if (player && selected) {
      const targetShip = client.scene.ships[selected];
      if (targetShip) {
        const playerPos = scalePosition({ x: player.x, y: player.y });
        const targetPos = scalePosition({ x: targetShip.x, y: targetShip.y });
        drawLine({
          x1: playerPos.x + 4, y1: playerPos.y + 4,
          x2: targetPos.x + 3, y2: targetPos.y + 3,
          color: "rgba(255, 50, 50, 0.7)",
          width: 2
        });
      }
    }

    // Path line
    if (client.scene?.isMoving) {
      const posPlayer = scalePosition({ x: player.x, y: player.y });
      const posFinal = scalePosition({ x: client.scene.targetx, y: client.scene.targety });
      drawLine({
        x1: posPlayer.x + 4,
        y1: posPlayer.y + 4,
        x2: posFinal.x + 4,
        y2: posFinal.y + 4,
        color: "rgba(255,255,255,0.68)",
        width: 1
      });
    }

    // Stats
    const fullStats = client.stats.getStats();
    const { pltPerHour, hnrPerHour, expPerHour, creditsPerHour } = fullStats;

    drawText({ text: "BTC:", x: 10, y: 250, })
    drawText({
      text: `${client.user.credits.toLocaleString()} (${creditsPerHour.toLocaleString()}/h)`,
      x: 50,
      y: 250,
    })

    drawText({ text: "PLT:", x: 10, y: 265, })
    drawText({ text: `${client.user.plt.toLocaleString()} (${pltPerHour.toLocaleString()}/h)`, x: 50, y: 265, })

    drawText({ text: "HNR:", x: 10, y: 280, })
    drawText({ text: `${client.user.honor.toLocaleString()} (${hnrPerHour.toLocaleString()}/h)`, x: 50, y: 280, })

    drawText({ text: "EXP:", x: 10, y: 295, })
    drawText({
      text: `${client.user.experience.toLocaleString()} (${expPerHour.toLocaleString()}/h)`,
      x: 50,
      y: 295,
    })

    // Kill/collect statistics
    const runHours = (Date.now() - client.stats.startTime) / (1000 * 60 * 60);
    const killsPerHour = runHours > 0 ? Math.round(client.stats.kills / runHours) : 0;
    const totalBoxes = client.stats.cargoBoxesCollected + client.stats.resourceBoxesCollected + client.stats.greenBoxesCollected;

    drawText({ text: "Kills:", x: 10, y: 310 })
    drawText({ text: `${client.stats.kills} (${killsPerHour}/h)`, x: 55, y: 310 })

    drawText({ text: "Deaths:", x: 10, y: 325 })
    drawText({ text: client.stats.deaths.toLocaleString(), x: 55, y: 325 })

    drawText({ text: "Boxes:", x: 10, y: 340 })
    drawText({ text: `${totalBoxes} (C:${client.stats.cargoBoxesCollected} B:${client.stats.resourceBoxesCollected} G:${client.stats.greenBoxesCollected})`, x: 55, y: 340 })

    drawText({ text: "Runtime:", x: 10, y: 355 })
    drawText({ text: fullStats.runTime, x: 55, y: 355 })

    // Bot state info panel (top-right)
    const panelX = canvas.width - 160;
    let panelY = 15;
    const lineH = 13;

    ctx.fillStyle = "rgba(0, 0, 0, 0.5)";
    ctx.fillRect(panelX - 5, 5, 160, 110);

    const sm = stateManager;
    if (sm) {
      const mode = client.controller?.kill ? "Kill" : client.controller?.collect ? "Collect" : client.controller?.killcollect ? "KillCollect" : "Follow";
      drawText({ text: `Mode: ${mode}`, x: panelX, y: panelY, size: 10 }); panelY += lineH;

      const killState = sm.kill?.attacking ? "Attacking" : sm.kill?.killInProgress ? "In Progress" : sm.kill?.enabled ? "Searching" : "Off";
      drawText({ text: `Kill: ${killState}`, x: panelX, y: panelY, size: 10, color: sm.kill?.attacking ? "#ff4444" : "#aaaaaa" }); panelY += lineH;

      const healthState = sm.detectors?.health?.lowHealthDetected ? "LOW" : sm.detectors?.health?.healthAdviced ? "Advised" : "OK";
      const healthColor = sm.detectors?.health?.lowHealthDetected ? "#ff4444" : sm.detectors?.health?.healthAdviced ? "#ffaa00" : "#44ff44";
      drawText({ text: `Health: ${healthState}`, x: panelX, y: panelY, size: 10, color: healthColor }); panelY += lineH;

      drawText({ text: `Escape: ${sm.escape?.enabled ? "Active" : "Off"}`, x: panelX, y: panelY, size: 10, color: sm.escape?.enabled ? "#ff8800" : "#aaaaaa" }); panelY += lineH;
      drawText({ text: `Recover: ${sm.recover?.enabled ? "Active" : "Off"}`, x: panelX, y: panelY, size: 10, color: sm.recover?.enabled ? "#44ff44" : "#aaaaaa" }); panelY += lineH;
      drawText({ text: `Orbiting: ${sm.navigation?.orbiting ? "Yes" : "No"}`, x: panelX, y: panelY, size: 10, color: sm.navigation?.orbiting ? "#00ccff" : "#aaaaaa" }); panelY += lineH;
      drawText({ text: `Following: ${sm.navigation?.following ? "Yes" : "No"}`, x: panelX, y: panelY, size: 10 }); panelY += lineH;

      const ammoNames = ["", "RLX_1", "GLX_2", "BLX_3", "GLX_2_AS", "MRS_6X"];
      const rocketNames = ["", "KEP_410", "NC_30", "TNC_130"];
      const ammo = ammoNames[sm.kill?.currentAmmo] || "-";
      const rocket = rocketNames[sm.kill?.currentRocket] || "-";
      drawText({ text: `Ammo: ${ammo} | Rkt: ${rocket}`, x: panelX, y: panelY, size: 10 });
    }

    requestAnimationFrame(render);
  }

  render();
});



