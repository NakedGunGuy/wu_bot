// Standalone bot runner for headless server deployment
// This runs a single bot instance without the web UI

require("dotenv").config();
const Client = require("./modules/GameClient");

// ========================
// CONFIGURE YOUR BOT HERE
// ========================

const BOT_CONFIG = {
  // Account credentials
  username: "miranjenkins",
  password: "okskmcr44",
  serverId: "na1",

  // Bot mode: "kill", "collect", "killcollect", or "follow"
  mode: "killcollect",

  // Bot settings
  settings: {
    workMap: "U-6",

    // NPCs to kill
    killTargets: [
      { name: "-=(Hydro)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Hyper|Hidro)=-", priority: 2, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Jenta)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Hyper|Jenta)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Hyper|Raider)=-", priority: 4, ammo: 2, rockets: 2, farmNearPortal: false },
      { name: "-=(Raider)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Bangoliour)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Hyper|Bangoliour)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Zavientos)=-", priority: 2, ammo: 2, rockets: 1, farmNearPortal: true },
      { name: "-=(Magmius)=-", priority: 2, ammo: 2, rockets: 1, farmNearPortal: true },
      { name: "-=(Hyper|Magmius)=-", priority: 4, ammo: 2, rockets: 1, farmNearPortal: true },
    ],

    // Boxes to collect - type: 0=bonus, 1=cargo, 3=green
    collectBoxTypes: [
      { type: 0, priority: 1 },
    ],

    // Health management
    minHP: 30,
    adviceHP: 70,

    // Kill settings
    kill: {
      targetEngagedNPC: false,
    },

    // Admin detection
    admin: {
      enabled: true,
      delay: 5,
    },

    // Escape settings
    escape: {
      enabled: true,
      delay: 20000,
    },

    // Config switching (when shields go down)
    config: {
      switchConfigOnShieldsDown: false,
      attacking: 1,
      fleeing: 2,
      flying: 2,
    },

    // Auto-enrichment
    // Laser materials: 0=Disabled, 4=Darkonit, 5=Uranit, 7=Dungid
    // Generator materials: 0=Disabled, 5=Uranit, 6=Azurit, 8=Xureon
    enrichment: {
      lasers: { enabled: true, materialType: 4 },
      rockets: { enabled: false, materialType: 0 },
      shields: { enabled: true, materialType: 5, amount: 10, minAmount: 10 },
      speed: { enabled: false, materialType: 0, amount: 10, minAmount: 10 },
    },

    // Auto-buying
    autobuy: {
      laser: {
        RLX_1: true,
        GLX_2: true,
        BLX_3: true,
        GLX_2_AS: false,
        MRS_6X: false,
      },
      rockets: {
        KEP_410: false,
        NC_30: true,
        TNC_130: true,
      },
      key: {
        enabled: false,
        savePLT: 50000,
      },
    },

    // Break schedule (anti-ban)
    break: {
      interval: 3600000, // 1 hour
      duration: 300000,  // 5 minutes
    },
  },
};

// ========================
// BOT INITIALIZATION
// ========================

console.log("╔════════════════════════════════════════╗");
console.log("║   War Universe Bot - Standalone Mode  ║");
console.log("╚════════════════════════════════════════╝");
console.log("");
console.log(`Username: ${BOT_CONFIG.username}`);
console.log(`Server:   ${BOT_CONFIG.serverId}`);
console.log(`Mode:     ${BOT_CONFIG.mode}`);
console.log(`Work Map: ${BOT_CONFIG.settings.workMap}`);
console.log("");
console.log("Initializing bot...");

const client = new Client({
  username: BOT_CONFIG.username,
  password: BOT_CONFIG.password,
  serverId: BOT_CONFIG.serverId,
});

// Apply settings
client.setSettings(BOT_CONFIG.settings);
client.setMode(BOT_CONFIG.mode);

// Event handlers
client.client.on("disconnected", () => {
  console.log("");
  console.log("❌ Client disconnected");
  console.log("Attempting to reconnect in 10 seconds...");

  setTimeout(() => {
    console.log("Reconnecting...");
    startBot();
  }, 10000);
});

// Stats logging
let statsInterval = null;

async function startBot() {
  try {
    await client.start();
    console.log("");
    console.log("✅ Bot started successfully!");
    console.log("");
    console.log("Press Ctrl+C to stop the bot");
    console.log("─────────────────────────────────────────");

    // Log stats every 30 seconds
    if (!statsInterval) {
      statsInterval = setInterval(() => {
        try {
          const stats = client.stats.getStats();
          console.log("");
          console.log(`[${new Date().toLocaleTimeString()}] Stats:`);
          console.log(`  HP: ${stats.hp}/${stats.maxHp} (${Math.round((stats.hp / stats.maxHp) * 100)}%)`);
          console.log(`  Shield: ${stats.shd}/${stats.maxShd} (${Math.round((stats.shd / stats.maxShd) * 100)}%)`);
          console.log(`  Position: ${stats.posX}, ${stats.posY}`);
          console.log(`  Map: ${stats.map}`);
          console.log(`  Credits: ${stats.credits?.toLocaleString() || 0}`);
          console.log(`  Uridium: ${stats.uridium?.toLocaleString() || 0}`);
        } catch (error) {
          // Ignore stats errors
        }
      }, 30000);
    }
  } catch (error) {
    console.error("");
    console.error("❌ Failed to start bot:");
    console.error(error.message);
    console.error("");
    console.error("Retrying in 30 seconds...");

    setTimeout(() => {
      startBot();
    }, 30000);
  }
}

// Handle graceful shutdown
process.on("SIGINT", () => {
  console.log("");
  console.log("Shutting down bot...");
  if (statsInterval) {
    clearInterval(statsInterval);
  }
  process.exit(0);
});

process.on("SIGTERM", () => {
  console.log("");
  console.log("Shutting down bot...");
  if (statsInterval) {
    clearInterval(statsInterval);
  }
  process.exit(0);
});

// Start the bot
startBot();
