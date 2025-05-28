const Client = require("./modules/GameClient.js");
const readline = require("readline");

const client = new Client({
  username: "",
  password: "",
  serverId: "",
});

// client.setSettings({
//   workMap: "R-6",
//   killTargets: [
//     { name: "-=(Hydro)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
//     { name: "-=(Hyper|Hidro)=-", priority: 2, ammo: 1, rockets: 1, farmNearPortal: false },
//     { name: "-=(Jenta)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
//     { name: "-=(Hyper|Jenta)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
//     { name: "-=(Hyper|Raider)=-", priority: 4, ammo: 2, rockets: 2, farmNearPortal: false },
//     { name: "-=(Raider)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
//     { name: "-=(Bangoliour)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
//     { name: "-=(Hyper|Bangoliour)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
//     { name: "-=(Zavientos)=-", priority: 2, ammo: 1, rockets: 1, farmNearPortal: true },
//     { name: "-=(Magmius)=-", priority: 2, ammo: 1, rockets: 1, farmNearPortal: true },
//     { name: "-=(Hyper|Magmius)=-", priority: 4, ammo: 1, rockets: 1, farmNearPortal: true },
//   ],
//   collectBoxTypes: [
//     { type: 3, priority: 10 },
//     // { type: 0, priority: 1 },
//     { type: 1, priority: 1 },
//   ],
//   minHP: 10,
//   adviceHP: 70,
//   kill: {
//     targetEngagedNPC: true,
//   },
//   escape: {
//     enabled: true,
//     delay: 20000,
//   },
//   config: {
//     switchConfigOnShieldsDown: true,
//     attacking: 1,
//     fleeing: 2,
//     flying: 2,
//   },
//   enrichment: {
//     // Laser enrichment materials: 0=Disabled, 4=Darkonit, 5=Uranit, 7=Dungid
//     lasers: { enabled: true, materialType: 4 }, // Using Darkonit
//     // Generator enrichment materials: 0=Disabled, 5=Uranit, 6=Azurit, 8=Xureon
//     rockets: { enabled: true, materialType: 6 }, // Using Azurit
//     shields: { enabled: true, materialType: 5, amount: 10, minAmount: 10 }, // Using Uranit
//     speed: { enabled: true, materialType: 8, amount: 10, minAmount: 10 }, // Using Xureon
//   },
//   autobuy: {
//     laser: {
//       RLX_1: true,
//       GLX_2: true,
//       BLX_3: true,
//       GLX_2_AS: true,
//       MRS_6X: true,
//     },
//     rockets: {
//       KEP_410: true,
//       NC_30: true,
//       TNC_130: true,
//     },
//     key: {
//       enabled: true,
//       savePLT: 50000,
//     },
//   },
//   break: {
//     interval: 0, // 1 hour
//     duration: 0, // 5 minutes
//   },
// });

client.setSettings({
  workMap: "U-6",
  killTargets: [
      { name: "-=(Hydro)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Hyper|Hidro)=-", priority: 2, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Jenta)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Hyper|Jenta)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Hyper|Raider)=-", priority: 4, ammo: 2, rockets: 2, farmNearPortal: false },
      { name: "-=(Raider)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Bangoliour)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Hyper|Bangoliour)=-", priority: 1, ammo: 1, rockets: 1, farmNearPortal: false },
      { name: "-=(Zavientos)=-", priority: 2, ammo: 1, rockets: 1, farmNearPortal: true },
      { name: "-=(Magmius)=-", priority: 2, ammo: 1, rockets: 1, farmNearPortal: true },
      { name: "-=(Hyper|Magmius)=-", priority: 4, ammo: 1, rockets: 1, farmNearPortal: true },
  ],
  collectBoxTypes: [{ type: 0, priority: 1 }], //type either 0 1 or 3. 3 is green box, I think 0 is bonus and 1 is resource box
  minHP: 10,
  adviceHP: 70,
  kill: {
    targetEngagedNPC: false,
  },
  admin: {
    enabled: true,
    delay: 5,
  },
  escape: {
    enabled: true,
    delay: 20000,
  },
  config: {
    switchConfigOnShieldsDown: false,
    attacking: 1,
    fleeing: 2,
    flying: 2,
  },
  enrichment: {
    // Laser enrichment materials: 0=Disabled, 4=Darkonit, 5=Uranit, 7=Dungid
    lasers: { enabled: false, materialType: 0 },
    // Generator enrichment materials: 0=Disabled, 5=Uranit, 6=Azurit, 8=Xureon
    rockets: { enabled: false, materialType: 0 },
    shields: { enabled: false, materialType: 0, amount: 10, minAmount: 10 },
    speed: { enabled: false, materialType: 0, amount: 10, minAmount: 10 },
  },
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
      TNC_130: false,
    },
    key: {
      enabled: false,
      savePLT: 50000,
    },
  },
  break: {
    interval: 3600000, // Break interval (1 hour)
    duration: 300000, // Break duration (5 minutes)
  },
});

client.setMode("killcollect"); //follow, kill, collect, killcollect. FOR FOLLOW SET A PLAYER ID SHIP IN modules/controllers/follow.js
client.start();

const rl = readline.createInterface({
  input: process.stdin,
  output: process.stdout,
});

//this is for manual control
rl.on("line", async (input) => {
  //client.client.sendPacket("ResourcesActionRequestPacket", {});
  if (input == "h") client.scene.move(15100, 1300); //send to home coords
  if (input === "") {
    const targetId = client.scene.ships[664111]?.selected;
    if (targetId) {
      console.log("targeting", targetId);

      await new Promise((resolve) => setTimeout(resolve, 100));
      client.client.sendPacket("UserActionsPacket", { actions: [{ actionId: 3 }] });
    }
  }
});
