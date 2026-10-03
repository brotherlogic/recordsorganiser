#!/bin/bash

# Ensure the 'recordsorganiser' session exists
if ! tmux has-session -t recordsorganiser 2>/dev/null; then
  # Create a new session named 'recordsorganiser', detached
  cd /workspaces/recordsorganiser
  tmux new-session -d -s recordsorganiser
  
  # Split the window horizontally (-h)
  # The left pane will remain a terminal
  # The right pane will run 'gh dash'
  tmux split-window -h -t recordsorganiser "gh dash"
fi
