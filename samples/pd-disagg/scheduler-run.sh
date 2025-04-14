#!/bin/bash

CONFIG_FILE="/etc/patio/instance-config.yaml"

# 监控文件是否存在，并执行相应操作
while true; do
  if [[ -f "$CONFIG_FILE" ]]; then
    # 执行scheduler 命令
    /app/scheduler/scheduler "$@" | tee -a /app/scheduler.log &

    scheduler_pid=$!

    # 获取文件的初始修改时间
    last_mod_time=$(stat -c %Y "$CONFIG_FILE")

    # 后台进程监控文件变更
    while true; do
      sleep 60

      # 获取当前文件的修改时间
      current_mod_time=$(stat -c %Y "$CONFIG_FILE")

      # 检查修改时间是否发生变化
      if [[ $current_mod_time -ne $last_mod_time ]]; then
        echo "File has changed, exiting script."
        kill $scheduler_pid
        exit 0
      fi
    done
  else
    echo "Config file not found. Waiting 5 seconds before rechecking."
    sleep 5
  fi
done