// GoChat Premium Client v2.7.0
const API_BASE = ''; 

class GoChatApp {
    constructor() {
        this.token = localStorage.getItem('token');
        this.user = JSON.parse(localStorage.getItem('user') || 'null');
        this.currentChat = null; // { conversation_id, peer_id, isGroup }
        this.conversations = [];
        this.friends = [];
        this.groups = [];
        this.messages = [];
        this.requests = [];
        this.knownUsers = {}; 
        this.knownGroups = {}; 
        
        this.currentView = 'chats'; 
        this.ws = null;
        this.reconnectAttempts = 0;
        this.heartbeatTimer = null;
        this.searchTimer = null;
        
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

        // Group
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
            this.searchTimer = setTimeout(() => this.handleSearch(e.target.value.trim()), 400);
        };
    }

    async request(path, options = {}) {
        const headers = options.headers || {};
        if (this.token) headers['Authorization'] = `Bearer ${this.token}`;
        if (options.body && !headers['Content-Type']) headers['Content-Type'] = 'application/json';

        const resp = await fetch(`${API_BASE}${path}`, { ...options, headers });
        let data = {};
        try { if (resp.status !== 204) data = await resp.json(); } catch(e) {}
        
        if (!resp.ok) {
            if (resp.status === 401) this.handleLogout();
            throw new Error(data.message || 'Request failed');
        }
        return data;
    }

    // --- Authentication ---
    async handleLogin() {
        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;
        const errorEl = document.getElementById('auth-error');
        try {
            const data = await this.request('/login', { method: 'POST', body: JSON.stringify({ username, password }) });
            this.token = data.token; this.user = data.user;
            localStorage.setItem('token', this.token);
            localStorage.setItem('user', JSON.stringify(this.user));
            this.showApp();
            await this.loadInitialData();
            this.connectWebSocket();
        } catch (err) { errorEl.textContent = err.message; errorEl.classList.remove('hidden'); }
    }

    async handleRegister() {
        const username = document.getElementById('reg-username').value;
        const nickname = document.getElementById('reg-nickname').value;
        const password = document.getElementById('reg-password').value;
        const errorEl = document.getElementById('auth-error');
        try {
            await this.request('/register', { method: 'POST', body: JSON.stringify({ username, nickname, password }) });
            alert('Registered! Please sign in.');
            this.switchAuthTab('login');
        } catch (err) { errorEl.textContent = err.message; errorEl.classList.remove('hidden'); }
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

    handleLogout() {
        this.stopHeartbeat();
        if (this.ws) this.ws.close();
        this.token = this.user = null;
        localStorage.clear();
        this.showAuth();
    }

    // --- UI Controls ---
    showApp() {
        document.getElementById('auth-page').classList.add('hidden');
        document.getElementById('app-page').classList.remove('hidden');
        this.updateMyProfile();
    }

    showAuth() {
        document.getElementById('app-page').classList.add('hidden');
        document.getElementById('auth-page').classList.remove('hidden');
        this.switchAuthTab('login');
    }

    switchAuthTab(type) {
        document.querySelectorAll('.auth-tab').forEach(tab => tab.classList.toggle('active', tab.dataset.type === type));
        document.getElementById('login-form').classList.toggle('hidden', type !== 'login');
        document.getElementById('register-form').classList.toggle('hidden', type !== 'register');
        document.getElementById('forgot-form').classList.add('hidden');
        document.getElementById('auth-error').classList.add('hidden');
    }

    showForgot() {
        document.getElementById('login-form').classList.add('hidden');
        document.getElementById('register-form').classList.add('hidden');
        document.getElementById('forgot-form').classList.remove('hidden');
    }

    switchView(view) {
        this.currentView = view;
        document.querySelectorAll('.nav-item').forEach(item => item.classList.toggle('active', item.dataset.view === view));
        document.getElementById('global-search').value = '';
        this.renderCurrentList();
    }

    showSettings() {
        document.getElementById('set-nickname').value = this.user.nickname;
        document.getElementById('set-avatar').value = this.user.avatar || '';
        document.getElementById('settings-modal').classList.remove('hidden');
    }

    async handleUpdateProfile() {
        const body = { nickname: document.getElementById('set-nickname').value, avatar: document.getElementById('set-avatar').value };
        try {
            const newUser = await this.request('/user/me', { method: 'PUT', body: JSON.stringify(body) });
            this.user = newUser;
            localStorage.setItem('user', JSON.stringify(this.user));
            this.updateMyProfile();
            document.getElementById('settings-modal').classList.add('hidden');
        } catch (err) { alert(err.message); }
    }

    updateMyProfile() {
        if (!this.user) return;
        document.getElementById('my-name').textContent = this.user.nickname;
        document.getElementById('my-avatar').textContent = (this.user.nickname || 'U')[0].toUpperCase();
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

    async onReceiveRealtimeMessage(msg) {
        if (msg.msg_type >= 10) return this.handleSignalMessage(msg);

        // Version Reconciliation
        if (msg.sender_info_version) {
            const cached = this.knownUsers[msg.sender_id];
            if (!cached || msg.sender_info_version > cached.version) {
                const u = await this.request(`/users/${msg.sender_id}`);
                this.knownUsers[u.id] = { nickname: u.nickname, avatar: u.avatar, version: u.info_version };
            }
        }

        if (msg.group_meta_version && msg.group_id) {
            const cached = this.knownGroups[msg.group_id];
            if (!cached || msg.group_meta_version > cached.version) {
                const g = await this.request(`/groups/${msg.group_id}`);
                this.knownGroups[g.id] = { name: g.name, avatar: g.avatar, version: g.meta_version };
                if (this.currentChat?.conversation_id === `group_${msg.group_id}`) {
                    document.getElementById('active-chat-name').textContent = g.name;
                }
            }
        }

        // Optimistic reconciliation
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

        if (msg.msg_type === 6) this.loadInitialData();
    }

    handleSignalMessage(msg) {
        switch (msg.msg_type) {
            case 10: this.loadRequests(); break;
            case 11: alert(`Friend request ${msg.content}`); this.loadRequests(); break;
            case 12: 
            case 13: 
                if (this.currentChat?.conversation_id === msg.conversation_id) {
                    alert(msg.msg_type === 12 ? 'Removed from group' : 'Group dismissed');
                    this.closeChat();
                }
                this.loadInitialData();
                break;
            case 14: this.loadInitialData(); break; // Identity Update
            case 15: this.loadFriends(); break; // Remark Update
        }
    }

    // --- Search & Interaction ---
    async handleSearch(keyword) {
        if (!keyword) return this.renderCurrentList();
        try {
            const data = await this.request(`/${this.currentView === 'groups' ? 'groups' : 'users'}/search?keyword=${encodeURIComponent(keyword)}`);
            const container = document.getElementById('list-content');
            if (this.currentView === 'groups') {
                container.innerHTML = (data.groups || []).map(g => `<div class="list-item">
                    <div class="avatar-circle">G</div>
                    <div class="list-item-info"><div class="list-item-name">${g.name}</div></div>
                    <button class="action-btn-small" onclick="app.restoreAndOpen('group_${g.id}', ${g.id}, true)">Join/Chat</button>
                </div>`).join('');
            } else {
                container.innerHTML = (data.users || []).map(u => `<div class="list-item">
                    <div class="avatar-circle">${(u.nickname || '?')[0].toUpperCase()}</div>
                    <div class="list-item-info"><div class="list-item-name">${u.nickname}</div></div>
                    ${this.user.id !== u.id ? `<button class="action-btn-small" onclick="app.handleApplyFriend(${u.id})">Add</button>` : ''}
                </div>`).join('');
            }
        } catch (e) {}
    }

    async handleApplyFriend(userId) {
        const message = prompt('Greeting:', 'Hi, I want to be your friend.');
        if (message !== null) await this.request('/friend/apply', { method: 'POST', body: JSON.stringify({ to_user_id: userId, message }) });
    }

    async handleJoinGroup(groupId) {
        const message = prompt('Intro:', 'Request to join');
        if (message !== null) {
            await this.request(`/groups/${groupId}/join`, { method: 'POST', body: JSON.stringify({ message }) });
            alert('Request sent!'); this.loadGroups();
        }
    }

    async handleHandleApply(applyId, accept) {
        await this.request('/friend/apply/handle', { method: 'POST', body: JSON.stringify({ apply_id: applyId, accept }) });
        await this.loadInitialData();
    }

    // --- Group Actions ---
    async handleCreateGroup(name) {
        const g = await this.request('/groups', { method: 'POST', body: JSON.stringify({ name }) });
        await this.loadInitialData(); this.openGroupChat(g.id);
    }

    async handleInviteMember() {
        if (!this.currentChat?.isGroup) return;
        try {
            const { members } = await this.request(`/groups/${this.currentChat.peer_id}/members`);
            const memberIds = new Set(members.map(m => m.user_id));
            const candidates = this.friends.filter(f => !memberIds.has(f.user_id) && f.user_id !== this.user.id);
            if (!candidates.length) return alert('No friends to invite.');
            const listEl = document.getElementById('invite-list');
            listEl.innerHTML = candidates.map(f => `
                <div class="member-item">
                    <input type="checkbox" class="invite-check" value="${f.user_id}" id="chk-${f.user_id}">
                    <label for="chk-${f.user_id}" style="margin-left:12px; cursor:pointer; flex:1; font-weight:600;">${f.nickname}</label>
                </div>`).join('');
            document.getElementById('invite-modal').classList.remove('hidden');
        } catch (e) { alert(e.message); }
    }

    async submitInvite() {
        const selected = Array.from(document.querySelectorAll('.invite-check:checked')).map(el => parseInt(el.value));
        if (!selected.length) return;
        await this.request(`/groups/${this.currentChat.peer_id}/invite`, { method: 'POST', body: JSON.stringify({ member_ids: selected }) });
        document.getElementById('invite-modal').classList.add('hidden');
        alert('Invited!');
    }

    async toggleMembers() {
        const panel = document.getElementById('member-list-panel');
        if (!panel.classList.contains('hidden')) return panel.classList.add('hidden');
        
        const data = await this.request(`/groups/${this.currentChat.peer_id}/members`);
        const group = this.groups.find(g => g.id === this.currentChat.peer_id);
        const isOwner = group?.owner_id === this.user.id;
        
        panel.innerHTML = data.members.map(m => {
            const name = m.nickname || `User ${m.user_id}`;
            const initial = name[0].toUpperCase();
            return `
            <div class="member-item">
                <div class="avatar-circle" style="width:34px; height:34px; font-size:12px; background: #cbd5e1;">${initial}</div>
                <div class="member-info-text">
                    <div class="member-name-row">
                        <span class="member-display-name">${name}</span>
                        ${m.user_id === this.user.id ? '<span class="member-role-tag role-me" onclick="app.showMemberSettings()" style="cursor:pointer;">You</span>' : ''}
                        ${m.role === 2 ? '<span class="member-role-tag role-owner">Owner</span>' : ''}
                    </div>
                    <div style="font-size: 11px; color: var(--text-muted);">ID: ${m.user_id}</div>
                </div>
                ${isOwner && m.user_id !== this.user.id ? `<button class="action-btn-small danger" onclick="app.handleKickMember(${m.user_id})">Kick</button>` : ''}
            </div>`;
        }).join('');
        panel.classList.remove('hidden');
    }

    async handleKickMember(userId) {
        if (confirm('Kick user?')) {
            await this.request(`/groups/${this.currentChat.peer_id}/kick/${userId}`, { method: 'POST' });
            this.toggleMembers(); 
        }
    }

    showMemberSettings() { document.getElementById('member-settings-modal').classList.remove('hidden'); }

    async handleUpdateGroupNickname() {
        const nickname = document.getElementById('set-group-nickname').value;
        await this.request(`/groups/${this.currentChat.peer_id}/nickname`, { method: 'PUT', body: JSON.stringify({ nickname }) });
        document.getElementById('member-settings-modal').classList.add('hidden');
        this.toggleMembers();
    }

    async handleUpdateAnnouncement() {
        const content = prompt('Announcement:');
        if (content) await this.request(`/groups/${this.currentChat.peer_id}/announcement`, { method: 'PUT', body: JSON.stringify({ content }) });
    }

    async handleQuitGroup() {
        if (confirm('Quit?')) {
            await this.request(`/groups/${this.currentChat.peer_id}/quit`, { method: 'POST' });
            this.closeChat();
        }
    }

    async handleDismissGroup() {
        if (confirm('DISMISS?')) {
            await this.request(`/groups/${this.currentChat.peer_id}`, { method: 'DELETE' });
            this.closeChat();
        }
    }

    closeChat() {
        this.currentChat = null;
        document.getElementById('chat-view').classList.add('hidden');
        document.getElementById('welcome-view').classList.remove('hidden');
        this.loadInitialData();
    }

    // --- Loading & Rendering ---
    async loadInitialData() {
        this.updateMyProfile();
        const [c, f, g, r] = await Promise.all([this.request('/conversations'), this.request('/friends'), this.request('/groups'), this.request('/friend/apply/list')]);
        this.conversations = c.conversations || [];
        this.friends = f.friends || [];
        this.groups = g.groups || [];
        this.requests = r.applies || [];
        this.updateBadge();
        this.renderCurrentList();
    }

    renderCurrentList() {
        if (this.currentView === 'chats') this.renderConversationList();
        else if (this.currentView === 'friends') this.renderFriendList();
        else if (this.currentView === 'groups') this.renderGroupList();
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
                ${c.unread_count > 0 ? `<span class="badge">${c.unread_count}</span>` : ''}
            </div>`).join('');
    }

    renderFriendList() {
        const pending = this.requests.filter(r => r.status === 0);
        let html = '';
        if (pending.length) {
            html += `<div style="padding:16px 20px; font-size:11px; font-weight:800; color:var(--text-muted); background:#f8fafc;">APPLICATIONS</div>`;
            html += pending.map(r => `<div class="list-item">
                <div class="avatar-circle">?</div>
                <div class="list-item-info"><div class="list-item-name">User ${r.from_user_id}</div><div class="list-item-preview">${r.message || 'Hi'}</div></div>
                <div style="display:flex; gap:10px;"><button class="accept-btn" onclick="app.handleHandleApply(${r.id}, true)">✔</button><button class="reject-btn" onclick="app.handleHandleApply(${r.id}, false)">✖</button></div>
            </div>`).join('');
        }
        html += `<div style="padding:16px 20px; font-size:11px; font-weight:800; color:var(--text-muted); background:#f8fafc;">FRIENDS</div>`;
        html += this.friends.map(f => `<div class="list-item" onclick="app.openPrivateChat(${f.user_id})">
            <div class="avatar-circle">${(f.nickname || 'U')[0].toUpperCase()}</div>
            <div class="list-item-info"><div class="list-item-name">${f.nickname}</div><div class="list-item-preview">ID: ${f.user_id}</div></div>
            <button class="action-btn-small" onclick="event.stopPropagation(); app.openPrivateChat(${f.user_id})">Chat</button>
        </div>`).join('');
        document.getElementById('list-content').innerHTML = html;
    }

    renderGroupList() {
        document.getElementById('list-content').innerHTML = this.groups.map(g => `<div class="list-item" onclick="app.openGroupChat(${g.id})">
            <div class="avatar-circle">G</div>
            <div class="list-item-info"><div class="list-item-name">${g.name}</div><div class="list-item-preview">Group ID: ${g.id}</div></div>
            <button class="action-btn-small" onclick="event.stopPropagation(); app.openGroupChat(${g.id})">Chat</button>
        </div>`).join('');
    }

    renderMessages() {
        document.getElementById('message-list').innerHTML = this.messages.map(m => `
            <div class="message-row ${m.sender_id === this.user.id ? 'self' : ''}">
                <div class="message-meta">${m.sender_id === this.user.id ? 'You' : 'User ' + m.sender_id}, ${this.formatTime(m.timestamp / 1000)}</div>
                <div class="message-bubble ${m.isOptimistic ? 'optimistic' : ''}">${m.content}</div>
            </div>`).join('');
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

    async openChat(id, pId, isG) {
        this.currentChat = { conversation_id: id, peer_id: pId, isGroup: isG };
        document.getElementById('welcome-view').classList.add('hidden');
        document.getElementById('chat-view').classList.remove('hidden');
        document.getElementById('member-list-panel').classList.add('hidden');
        
        const avatarEl = document.getElementById('active-avatar');
        avatarEl.textContent = isG ? 'G' : 'U';
        document.getElementById('active-chat-name').textContent = isG ? `Group: ${pId}` : `User: ${pId}`;
        
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
        
        const data = await this.request(`/messages?conversation_id=${id}`);
        this.messages = data.messages || []; this.renderMessages(); this.scrollToBottom();
        this.request('/conversations/clear_unread', { method: 'POST', body: JSON.stringify({ conversation_id: id }) }).then(() => this.loadConversations());
    }

    async restoreAndOpen(id, pId, isG) {
        try {
            await this.request('/conversations/restore', { method: 'POST', body: JSON.stringify({ conversation_id: id }) });
            this.switchView('chats'); this.openChat(id, pId, isG);
        } catch(e) { this.openChat(id, pId, isG); }
    }

    openPrivateChat(userId) { const id = this.user.id < userId ? `conv_${this.user.id}_${userId}` : `conv_${userId}_${this.user.id}`; this.restoreAndOpen(id, userId, false); }
    openGroupChat(groupId) { this.restoreAndOpen(`group_${groupId}`, groupId, true); }
    updateBadge() { const badge = document.getElementById('request-badge'); const count = this.requests.filter(r => r.status === 0).length; badge.textContent = count; badge.classList.toggle('hidden', count === 0); }
    formatTime(ts) { if (!ts) return ''; const d = new Date(ts * 1000); return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }); }
    scrollToBottom() { const el = document.getElementById('message-list'); el.scrollTop = el.scrollHeight; }
}

const app = new GoChatApp();
