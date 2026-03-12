// GoChat Client v2.4.0
const API_BASE = ''; 

class GoChatApp {
    constructor() {
        this.token = localStorage.getItem('token');
        this.user = JSON.parse(localStorage.getItem('user') || 'null');
        this.currentChat = null;
        this.conversations = [];
        this.friends = [];
        this.groups = [];
        this.messages = [];
        this.requests = [];
        this.currentView = 'chats';
        this.ws = null;
        this.reconnectAttempts = 0;
        this.heartbeatTimer = null;
        this.init();
    }

    async init() {
        this.bindEvents();
        if (this.token) {
            try {
                const me = await this.request('/user/me');
                this.user = me;
                localStorage.setItem('user', JSON.stringify(this.user));
                this.showApp();
                await this.loadInitialData();
                this.connectWebSocket();
            } catch (err) { this.handleLogout(); }
        } else this.showAuth();
    }

    bindEvents() {
        // Auth
        document.getElementById('login-btn').onclick = () => this.handleLogin();
        document.getElementById('register-btn').onclick = () => this.handleRegister();
        document.getElementById('logout-btn').onclick = () => this.handleLogout();
        document.getElementById('reset-btn').onclick = () => this.handleForgotPassword();
        
        document.querySelectorAll('.auth-tab').forEach(tab => tab.onclick = () => this.switchAuthTab(tab.dataset.type));
        document.querySelectorAll('.nav-item').forEach(item => item.onclick = () => this.switchView(item.dataset.view));

        // Messaging
        document.getElementById('send-msg-btn').onclick = () => this.handleSendMessage();
        document.getElementById('chat-input').onkeypress = (e) => { if (e.key === 'Enter') this.handleSendMessage(); };

        // Group Actions
        document.getElementById('create-group-btn').onclick = () => {
            const name = prompt('Enter group name:');
            if (name) this.handleCreateGroup(name);
        };
        document.getElementById('invite-btn').onclick = () => this.handleInviteMember();
        document.getElementById('members-btn').onclick = () => this.toggleMembers();
        document.getElementById('quit-group-btn').onclick = () => this.handleQuitGroup();
        document.getElementById('dismiss-group-btn').onclick = () => this.handleDismissGroup();

        // Search
        document.getElementById('global-search').oninput = (e) => {
            clearTimeout(this.searchTimer);
            this.searchTimer = setTimeout(() => this.handleSearch(e.target.value.trim()), 500);
        };
    }

    async request(path, options = {}) {
        const headers = options.headers || {};
        if (this.token) headers['Authorization'] = `Bearer ${this.token}`;
        if (options.body && !headers['Content-Type']) headers['Content-Type'] = 'application/json';

        const resp = await fetch(`${API_BASE}${path}`, { ...options, headers });
        const data = await resp.json();
        if (!resp.ok) {
            if (resp.status === 401) this.handleLogout();
            throw new Error(data.message || 'Request failed');
        }
        return data;
    }

    // --- Profile & Auth ---
    async handleLogin() {
        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;
        try {
            const data = await this.request('/login', { method: 'POST', body: JSON.stringify({ username, password }) });
            this.token = data.token;
            this.user = data.user;
            localStorage.setItem('token', this.token);
            localStorage.setItem('user', JSON.stringify(this.user));
            this.showApp();
            await this.loadInitialData();
            this.connectWebSocket();
        } catch (err) { alert(err.message); }
    }

    async handleRegister() {
        const username = document.getElementById('reg-username').value;
        const nickname = document.getElementById('reg-nickname').value;
        const password = document.getElementById('reg-password').value;
        try {
            await this.request('/register', { method: 'POST', body: JSON.stringify({ username, nickname, password }) });
            this.switchAuthTab('login');
            alert('Registered! Please login.');
        } catch (err) { alert(err.message); }
    }

    async handleForgotPassword() {
        const username = document.getElementById('forgot-username').value;
        const new_password = document.getElementById('forgot-new-password').value;
        try {
            await this.request('/forgot_password', { method: 'POST', body: JSON.stringify({ username, new_password }) });
            alert('Password reset successful!');
            this.switchAuthTab('login');
        } catch (err) { alert(err.message); }
    }

    showForgot() {
        document.getElementById('login-form').classList.add('hidden');
        document.getElementById('forgot-form').classList.remove('hidden');
    }

    showSettings() {
        document.getElementById('set-nickname').value = this.user.nickname;
        document.getElementById('set-avatar').value = this.user.avatar || '';
        document.getElementById('set-phone').value = this.user.phone || '';
        document.getElementById('set-email').value = this.user.email || '';
        document.getElementById('settings-modal').classList.remove('hidden');
    }

    async handleUpdateProfile() {
        const body = {
            nickname: document.getElementById('set-nickname').value,
            avatar: document.getElementById('set-avatar').value,
            phone: document.getElementById('set-phone').value,
            email: document.getElementById('set-email').value
        };
        try {
            const newUser = await this.request('/user/me', { method: 'PUT', body: JSON.stringify(body) });
            this.user = newUser;
            localStorage.setItem('user', JSON.stringify(this.user));
            this.updateMyProfile();
            document.getElementById('settings-modal').classList.add('hidden');
        } catch (err) { alert(err.message); }
    }

    handleLogout() {
        this.stopHeartbeat();
        if (this.ws) this.ws.close();
        this.token = this.user = null;
        localStorage.clear();
        this.showAuth();
    }

    // --- Real-time Logic ---
    connectWebSocket() {
        if (this.ws) this.ws.close();
        const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/ws?token=${this.token}`;
        this.ws = new WebSocket(wsUrl);

        this.ws.onopen = () => { this.reconnectAttempts = 0; this.startHeartbeat(); };
        this.ws.onmessage = (event) => {
            if (event.data === 'pong') return;
            try { this.onReceiveRealtimeMessage(JSON.parse(event.data)); } catch (e) {}
        };
        this.ws.onclose = () => {
            this.stopHeartbeat();
            if (this.token) setTimeout(() => { this.reconnectAttempts++; this.connectWebSocket(); }, Math.min(1000 * Math.pow(2, this.reconnectAttempts), 30000));
        };
    }

    startHeartbeat() {
        this.stopHeartbeat();
        this.heartbeatTimer = setInterval(() => { if (this.ws?.readyState === WebSocket.OPEN) this.ws.send('ping'); }, 30000);
    }
    stopHeartbeat() { if (this.heartbeatTimer) clearInterval(this.heartbeatTimer); }

    onReceiveRealtimeMessage(msg) {
        // --- SIGNAL HANDLING (NON-DISPLAY) ---
        if (msg.msg_type >= 10) {
            this.handleSignalMessage(msg);
            return;
        }

        // --- DISPLAY MESSAGES ---
        const existingIdx = this.messages.findIndex(m => m.msg_id === msg.msg_id || (m.isOptimistic && m.content === msg.content));
        
        if (this.currentChat?.conversation_id === msg.conversation_id) {
            if (existingIdx !== -1) this.messages[existingIdx] = { ...msg, isOptimistic: false };
            else { this.messages.push(msg); this.scrollToBottom(); }
            this.renderMessages();
        }

        let conv = this.conversations.find(c => c.conversation_id === msg.conversation_id);
        if (conv) {
            conv.last_message = msg.content;
            conv.last_message_time = msg.timestamp / 1000;
            if (!this.currentChat || this.currentChat.conversation_id !== msg.conversation_id) conv.unread_count++;
            this.renderConversationList();
        } else this.loadConversations();

        if (msg.msg_type === 6) this.loadInitialData(); // Trigger reload on system messages
    }

    handleSignalMessage(msg) {
        console.log('Received signal:', msg.msg_type, msg.content);
        switch (msg.msg_type) {
            case 10: // Friend Apply
                this.loadRequests();
                break;
            case 11: // Friend Reject
                alert(`Friend request ${msg.content}`);
                this.loadRequests();
                break;
            case 12: // Kicked / Left Group
                if (this.currentChat?.conversation_id === msg.conversation_id) {
                    alert('You have been removed from this group.');
                    this.currentChat = null;
                    document.getElementById('chat-view').classList.add('hidden');
                    document.getElementById('welcome-view').classList.remove('hidden');
                }
                this.loadInitialData();
                break;
            case 13: // Group Dismissed
                alert('Group has been dismissed by owner.');
                if (this.currentChat?.conversation_id === msg.conversation_id) {
                    this.currentChat = null;
                    document.getElementById('chat-view').classList.add('hidden');
                    document.getElementById('welcome-view').classList.remove('hidden');
                }
                this.loadInitialData();
                break;
        }
    }

    // --- Search & Groups ---
    async handleSearch(keyword) {
        if (!keyword) { this.switchView(this.currentView); return; }
        try {
            const data = await this.request(`/${this.currentView === 'groups' ? 'groups' : 'users'}/search?keyword=${encodeURIComponent(keyword)}`);
            const container = document.getElementById('list-content');
            if (this.currentView === 'groups') {
                container.innerHTML = (data.groups || []).map(g => `<div class="list-item">
                    <div class="avatar-circle">G</div>
                    <div class="list-item-info"><div class="list-item-name">${g.name}</div></div>
                    <button class="action-btn-small" onclick="app.handleJoinGroup(${g.id})">Join</button>
                </div>`).join('');
            } else {
                container.innerHTML = (data.users || []).map(u => `<div class="list-item">
                    <div class="avatar-circle">${u.nickname[0]}</div>
                    <div class="list-item-info"><div class="list-item-name">${u.nickname}</div></div>
                    ${this.user.id !== u.id ? `<button class="action-btn-small" onclick="app.handleApplyFriend(${u.id})">Add</button>` : ''}
                </div>`).join('');
            }
        } catch (e) {}
    }

    async handleJoinGroup(groupId) {
        const message = prompt('Message:');
        try {
            await this.request(`/groups/${groupId}/join`, { method: 'POST', body: JSON.stringify({ message }) });
            alert('Joined!'); this.loadGroups();
        } catch (err) { alert(err.message); }
    }

    async handleCreateGroup(name) {
        try {
            const g = await this.request('/groups', { method: 'POST', body: JSON.stringify({ name }) });
            await this.loadGroups(); this.openGroupChat(g.id);
        } catch (err) { alert(err.message); }
    }

    async handleInviteMember() {
        if (!this.currentChat?.isGroup) return;
        try {
            // 1. Get current members
            const { members } = await this.request(`/groups/${this.currentChat.peer_id}/members`);
            const memberIds = new Set(members.map(m => m.user_id));
            
            // 2. Filter friends who are not members
            const invitees = this.friends.filter(f => !memberIds.has(f.user_id));
            
            if (invitees.length === 0) return alert('All your friends are already in this group.');

            // 3. Show Modal
            const listEl = document.getElementById('invite-list');
            listEl.innerHTML = invitees.map(f => `
                <div class="member-item">
                    <input type="checkbox" class="invite-check" value="${f.user_id}" id="chk-${f.user_id}">
                    <label for="chk-${f.user_id}" style="margin-left: 10px; cursor: pointer;">${f.nickname}</label>
                </div>
            `).join('');
            document.getElementById('invite-modal').classList.remove('hidden');
        } catch (e) { alert(e.message); }
    }

    async submitInvite() {
        const selected = Array.from(document.querySelectorAll('.invite-check:checked')).map(el => parseInt(el.value));
        if (selected.length === 0) return;
        try {
            await this.request(`/groups/${this.currentChat.peer_id}/invite`, { method: 'POST', body: JSON.stringify({ member_ids: selected }) });
            document.getElementById('invite-modal').classList.add('hidden');
            alert('Invited!');
        } catch (e) { alert(e.message); }
    }

    async toggleMembers() {
        const panel = document.getElementById('member-list-panel');
        if (!panel.classList.contains('hidden')) return panel.classList.add('hidden');
        try {
            const data = await this.request(`/groups/${this.currentChat.peer_id}/members`);
            const group = this.groups.find(g => g.id === this.currentChat.peer_id);
            const isOwner = group?.owner_id === this.user.id;
            panel.innerHTML = data.members.map(m => `
                <div class="member-item">
                    <div class="avatar-circle" style="width: 32px; height: 32px; font-size: 11px;">${(m.nickname || '?')[0]}</div>
                    <div class="member-info-text">
                        <div class="member-name-row">
                            <span style="overflow: hidden; text-overflow: ellipsis;">${m.nickname || 'User ' + m.user_id}</span>
                            ${m.user_id === this.user.id ? '<span class="member-role-tag role-me">You</span>' : ''}
                            ${m.role === 2 ? '<span class="member-role-tag role-owner">Owner</span>' : ''}
                        </div>
                    </div>
                    ${isOwner && m.user_id !== this.user.id ? `<button class="action-btn-small danger" onclick="app.handleKickMember(${m.user_id})">Kick</button>` : ''}
                </div>
            `).join('');
            panel.classList.remove('hidden');
        } catch (e) {}
    }

    async handleKickMember(userId) {
        if (confirm('Kick user?')) {
            await this.request(`/groups/${this.currentChat.peer_id}/kick/${userId}`, { method: 'POST' });
            this.toggleMembers(); 
        }
    }

    async handleUpdateAnnouncement() {
        const content = prompt('Announcement:');
        if (content) await this.request(`/groups/${this.currentChat.peer_id}/announcement`, { method: 'PUT', body: JSON.stringify({ content }) });
    }

    async handleQuitGroup() {
        if (confirm('Quit?')) {
            await this.request(`/groups/${this.currentChat.peer_id}/quit`, { method: 'POST' });
            this.currentChat = null; this.switchView('chats'); this.loadInitialData();
            document.getElementById('chat-view').classList.add('hidden');
            document.getElementById('welcome-view').classList.remove('hidden');
        }
    }

    async handleDismissGroup() {
        if (confirm('DISMISS?')) {
            await this.request(`/groups/${this.currentChat.peer_id}`, { method: 'DELETE' });
            this.currentChat = null; this.switchView('chats'); this.loadInitialData();
            document.getElementById('chat-view').classList.add('hidden');
            document.getElementById('welcome-view').classList.remove('hidden');
        }
    }

    // --- Loading & Rendering ---
    async loadInitialData() {
        this.updateMyProfile();
        const [c, f, g, r] = await Promise.all([
            this.request('/conversations'), this.request('/friends'), this.request('/groups'), this.request('/friend/apply/list')
        ]);
        this.conversations = c.conversations || [];
        this.friends = f.friends || [];
        this.groups = g.groups || [];
        this.requests = r.applies || [];
        this.updateBadge();
        this.switchView(this.currentView);
    }

    async loadMessages(id) {
        const data = await this.request(`/messages?conversation_id=${id}`);
        this.messages = data.messages || [];
        this.renderMessages(); this.scrollToBottom();
    }

    renderConversationList() {
        const sorted = [...this.conversations].sort((a, b) => (b.is_top - a.is_top) || (b.last_message_time - a.last_message_time));
        document.getElementById('list-content').innerHTML = sorted.map(c => `
            <div class="list-item ${this.currentChat?.conversation_id === c.conversation_id ? 'active' : ''}" 
                 onclick="app.openChat('${c.conversation_id}', ${c.peer_id}, ${c.conversation_id.startsWith('group_')})">
                <div class="avatar-circle">${c.conversation_id.startsWith('group') ? 'G' : 'U'}</div>
                <div class="list-item-info">
                    <div class="list-item-title">
                        <span class="list-item-name">${c.conversation_id.startsWith('group') ? 'Group' : 'User'} ${c.peer_id}</span>
                        <span class="list-item-time">${this.formatTime(c.last_message_time)}</span>
                    </div>
                    <div class="list-item-preview">${c.last_message || '...'}</div>
                </div>
                ${c.unread_count > 0 ? `<span class="badge ${c.is_muted ? 'muted' : ''}">${c.unread_count}</span>` : ''}
            </div>
        `).join('');
    }

    renderFriendList() {
        const pending = this.requests.filter(r => r.status === 0);
        let html = '';
        if (pending.length) {
            html += `<div style="padding:10px 20px; font-size:11px; font-weight:700; color:var(--text-muted);">APPLICATIONS</div>`;
            html += pending.map(r => `
                <div class="list-item">
                    <div class="avatar-circle">?</div>
                    <div class="list-item-info"><div class="list-item-name">User ${r.from_user_id}</div></div>
                    <div style="display:flex; gap:8px;">
                        <button class="accept-btn" onclick="app.handleHandleApply(${r.id}, true)">✔</button>
                        <button class="reject-btn" onclick="app.handleHandleApply(${r.id}, false)">✖</button>
                    </div>
                </div>
            `).join('');
        }
        html += `<div style="padding:10px 20px; font-size:11px; font-weight:700; color:var(--text-muted);">FRIENDS</div>`;
        html += this.friends.map(f => `
            <div class="list-item" onclick="app.openPrivateChat(${f.user_id})">
                <div class="avatar-circle">${(f.nickname || 'U')[0]}</div>
                <div class="list-item-info">
                    <div class="list-item-name">${f.nickname}</div>
                    <div class="list-item-preview">ID: ${f.user_id}</div>
                </div>
                <button class="action-btn-small" onclick="event.stopPropagation(); app.openPrivateChat(${f.user_id})">Chat</button>
            </div>
        `).join('');
        document.getElementById('list-content').innerHTML = html;
    }

    renderGroupList() {
        document.getElementById('list-content').innerHTML = this.groups.map(g => `
            <div class="list-item" onclick="app.openGroupChat(${g.id})">
                <div class="avatar-circle">G</div>
                <div class="list-item-info">
                    <div class="list-item-name">${g.name}</div>
                    <div class="list-item-preview">${g.description || '...'}</div>
                </div>
                <button class="action-btn-small" onclick="event.stopPropagation(); app.openGroupChat(${g.id})">Chat</button>
            </div>
        `).join('');
    }

    renderMessages() {
        document.getElementById('message-list').innerHTML = this.messages.map(m => `
            <div class="message-row ${m.sender_id === this.user.id ? 'self' : ''}">
                <div class="message-meta">${m.sender_id === this.user.id ? 'You' : 'User ' + m.sender_id}, ${this.formatTime(m.timestamp / 1000)}</div>
                <div class="message-bubble ${m.isOptimistic ? 'optimistic' : ''}">${m.content}</div>
            </div>
        `).join('');
    }

    async handleSendMessage() {
        const input = document.getElementById('chat-input');
        const content = input.value.trim();
        if (!content || !this.currentChat) return;
        const opt = { msg_id: 'opt_' + Date.now(), conversation_id: this.currentChat.conversation_id, sender_id: this.user.id, content, timestamp: Date.now(), isOptimistic: true };
        this.messages.push(opt); this.renderMessages(); this.scrollToBottom(); input.value = '';
        try {
            const body = { conversation_id: this.currentChat.conversation_id, content, msg_type: 1 };
            if (this.currentChat.isGroup) body.group_id = this.currentChat.peer_id;
            else body.receiver_id = this.currentChat.peer_id;
            await this.request('/messages/send', { method: 'POST', body: JSON.stringify(body) });
        } catch (e) { this.messages = this.messages.filter(m => m.msg_id !== opt.msg_id); this.renderMessages(); alert(e.message); }
    }

    openChat(id, pId, isG) {
        this.currentChat = { conversation_id: id, peer_id: pId, isGroup: isG };
        document.getElementById('welcome-view').classList.add('hidden');
        document.getElementById('chat-view').classList.remove('hidden');
        document.getElementById('member-list-panel').classList.add('hidden');
        document.getElementById('active-chat-name').textContent = isG ? `Group ${pId}` : `User ${pId}`;
        const groupActions = document.getElementById('group-actions');
        if (isG) {
            groupActions.classList.remove('hidden');
            const group = this.groups.find(g => g.id === pId);
            const isOwner = group?.owner_id === this.user.id;
            document.getElementById('dismiss-group-btn').classList.toggle('hidden', !isOwner);
            document.getElementById('quit-group-btn').classList.toggle('hidden', isOwner);
            document.getElementById('active-chat-name').onclick = isOwner ? () => this.handleUpdateAnnouncement() : null;
            document.getElementById('active-chat-name').style.cursor = isOwner ? 'pointer' : 'default';
        } else groupActions.classList.add('hidden');
        this.loadMessages(id);
        this.request('/conversations/clear_unread', { method: 'POST', body: JSON.stringify({ conversation_id: id }) }).then(() => this.loadConversations());
    }

    openPrivateChat(userId) { this.switchView('chats'); this.openChat(this.user.id < userId ? `conv_${this.user.id}_${userId}` : `conv_${userId}_${this.user.id}`, userId, false); }
    openGroupChat(groupId) { this.switchView('chats'); this.openChat(`group_${groupId}`, groupId, true); }

    switchView(view) {
        this.currentView = view;
        document.querySelectorAll('.nav-item').forEach(item => item.classList.toggle('active', item.dataset.view === view));
        document.getElementById('global-search').value = '';
        if (view === 'chats') this.renderConversationList();
        else if (view === 'friends') this.renderFriendList();
        else if (view === 'groups') this.renderGroupList();
    }

    switchAuthTab(type) {
        document.querySelectorAll('.auth-tab').forEach(tab => tab.classList.toggle('active', tab.dataset.type === type));
        document.getElementById('login-form').classList.toggle('hidden', type !== 'login');
        document.getElementById('register-form').classList.toggle('hidden', type !== 'register');
        document.getElementById('forgot-form').classList.add('hidden');
    }

    showApp() { document.getElementById('auth-page').classList.add('hidden'); document.getElementById('app-page').classList.remove('hidden'); }
    showAuth() { document.getElementById('app-page').classList.add('hidden'); document.getElementById('auth-page').classList.remove('hidden'); }
    updateMyProfile() { if (!this.user) return; document.getElementById('my-name').textContent = this.user.nickname; document.getElementById('my-avatar').textContent = (this.user.nickname || 'U')[0]; }
    updateBadge() { const badge = document.getElementById('request-badge'); const count = this.requests.filter(r => r.status === 0).length; badge.textContent = count; badge.classList.toggle('hidden', count === 0); }
    getAvatarChar(id) { return id.startsWith('group') ? 'G' : 'U'; }
    formatTime(ts) { if (!ts) return ''; const date = new Date(ts * 1000); return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }); }
    scrollToBottom() { const el = document.getElementById('message-list'); el.scrollTop = el.scrollHeight; }
}

const app = new GoChatApp();
