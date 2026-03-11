    onReceiveRealtimeMessage(msg) {
        // 1. Add to messages if current chat matches
        if (this.currentChat && this.currentChat.conversation_id === msg.conversation_id) {
            // Check if msg already exists (to avoid duplicates from local sending)
            if (!this.messages.find(m => m.msg_id === msg.msg_id)) {
                this.messages.push({
                    msg_id: msg.msg_id,
                    sender_id: msg.sender_id,
                    content: msg.content,
                    timestamp: msg.timestamp,
                    conversation_id: msg.conversation_id
                });
                this.renderMessages();
            }
        }

        // 2. Update conversation list preview
        let conv = this.conversations.find(c => c.conversation_id === msg.conversation_id);
        if (conv) {
            conv.last_message = msg.content;
            conv.last_message_time = msg.timestamp / 1000;
            // If it's not the current chat, increment unread count
            if (!this.currentChat || this.currentChat.conversation_id !== msg.conversation_id) {
                conv.unread_count++;
            }
        } else {
            // New conversation
            this.loadConversations(); // Reload all to get the new entry
        }
        this.renderConversationList();
    }

    startHeartbeat() {
        this.stopHeartbeat();
        this.heartbeatTimer = setInterval(() => {
            if (this.ws && this.ws.readyState === WebSocket.OPEN) {
                this.ws.send('ping');
            }
        }, 30000); // 30 seconds
    }

    stopHeartbeat() {
        if (this.heartbeatTimer) {
            clearInterval(this.heartbeatTimer);
            this.heartbeatTimer = null;
        }
    }
