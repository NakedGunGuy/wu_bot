module.exports = {
  apps: [
    {
      name: "wu-bot",
      script: "./bot.js",  // Standalone bot runner
      instances: 1,
      exec_mode: "fork",
      autorestart: true,
      watch: false,
      max_memory_restart: "500M",
      env: {
        NODE_ENV: "production",
      },
      error_file: "./logs/err.log",
      out_file: "./logs/out.log",
      log_file: "./logs/combined.log",
      time: true,
      merge_logs: true,
      log_date_format: "YYYY-MM-DD HH:mm:ss Z",
      // Restart configuration
      min_uptime: "10s",
      max_restarts: 10,
      restart_delay: 4000,
      // Kill timeout
      kill_timeout: 5000,
      // Listen timeout
      listen_timeout: 3000,
    },
  ],
};
