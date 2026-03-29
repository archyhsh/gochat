// GoChat Premium Client v2.8.2 - Master Build Final Correction
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
        this.requests = []; // Friend requests
        this.groupRequests = []; // Inbound group join requests
        
        // Cache for versioning (Identity Management)
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
        // --- Authentication ---
        document.getElementById('login-btn').onclick = () => this.handleLogin();
        document.getElementById('register-btn').onclick = () => this.handleRegister();
        document.getElementById('logout-btn').onclick = () => this.handleLogout();
        document.getElementById('reset-btn').onclick = () => this.handleForgotPassword();
        
        document.querySelectorAll('.auth-tab').forEach(tab => tab.onclick = () => this.switchAuthTab(tab.dataset.type));
        document.querySelectorAll('.nav-item').forEach(item => item.onclick = () => this.switchView(item.dataset.view));

        // --- Chat Controls ---
        document.getElementById('send-msg-btn').onclick = () => this.handleSendMessage();
        document.getElementById('chat-input').onkeypress = (e) => {
            if (e.key === 'Enter') this.handleSendMessage();
        };

        // --- Group Actions ---
        document.getElementById('create-group-btn').onclick = () => {
            const name = prompt('Enter group name:');
            if (name) this.handleCreateGroup(name);
        };
        document.getElementById('invite-btn').onclick = () => this.handleInviteMember();
        document.getElementById('members-btn').onclick = () => this.toggleMembers();
        document.getElementById('quit-group-btn').onclick = () => this.handleQuitGroup();
        document.getElementById('dismiss-group-btn').onclick = () => this.handleDismissGroup();

        // --- Friend Actions ---
        document.getElementById('block-friend-btn').onclick = () => this.handleBlockFriend();
        document.getElementById('delete-friend-btn').onclick = () => this.handleDeleteFriend();

        // --- Search ---
        document.getElementById('global-search').oninput = (e) => {
            clearTimeout(this.searchTimer);
            const val = e.target.value.trim();
            this.searchTimer = setTimeout(() => this.handleSearch(val), 400);
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

    // --- UI View Control ---
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
        const searchInput = document.getElementById('global-search');
        
        // 1. DYNAMIC SEARCH PLACEHOLDERS
        const placeholders = {
            'chats': 'Search chats by remark or group name...',
            'friends': 'Find new users by nickname...',
            'groups': 'Discover public groups by name...'
        };
        searchInput.placeholder = placeholders[view] || 'Search...';
        searchInput.value = '';

        document.querySelectorAll('.nav-item').forEach(item => item.classList.toggle('active', item.dataset.view === view));
        this.renderCurrentList();
    }

    // --- Search Logic ---
    async handleSearch(keyword) {
        if (!keyword) return this.renderCurrentList();
        try {
            const path = this.currentView === 'chats' ? `/conversations?keyword=${encodeURIComponent(keyword)}` :
                         this.currentView === 'friends' ? `/users/search?keyword=${encodeURIComponent(keyword)}` :
                         `/groups/search?keyword=${encodeURIComponent(keyword)}`;
            const data = await this.request(path);
            this.renderSearchResults(data.conversations || data.users || data.groups || data.Groups || [], this.currentView);
        } catch (e) { console.error('Search error:', e); }
    }

    renderSearchResults(results, context) {
        const container = document.getElementById('list-content');
        // SECURITY FILTER: Remove self
        const filtered = results.filter(item => {
            const id = item.id || item.peer_id || item.group_id || item.GroupId;
            return id !== this.user.id;
        });

        if (!filtered.length) return container.innerHTML = `<div class="empty-state">No matching results</div>`;
        
        container.innerHTML = filtered.map(item => {
            if (context === 'chats') return `
                <div class="list-item" onclick="app.restoreAndOpen('${item.conversation_id}', ${item.peer_id}, ${item.conversation_id.startsWith('group_')})">
                    <div class="avatar-circle">${item.conversation_id.startsWith('group') ? 'G' : 'U'}</div>
                    <div class="list-item-info">
                        <div class="list-item-name">${item.conversation_id}</div>
                        <div class="list-item-preview">Click to restore/open</div>
                    </div>
                </div>`;
            
            if (context === 'friends') {
                const isFriend = this.friends.some(f => f.user_id === item.id);
                return `
                <div class="list-item">
                    <div class="avatar-circle">${(item.nickname || '?')[0].toUpperCase()}</div>
                    <div class="list-item-info"><div class="list-item-name">${item.nickname}</div></div>
                    ${isFriend ? `<span class="member-role-tag role-me" style="background:#cbd5e1; color:#475569;">Friend</span>` : `<button class="action-btn-small" onclick="app.handleApplyFriend(${item.id})">Add</button>`}
                </div>`;
            }

            const gId = item.id || item.group_id || item.GroupId;
            const name = item.name || item.Name;
            const isJoined = this.groups.some(g => (g.id === gId || g.group_id === gId));
            return `
                <div class="list-item">
                    <div class="avatar-circle">G</div>
                    <div class="list-item-info"><div class="list-item-name">${name}</div></div>
                    ${isJoined ? `<span class="member-role-tag role-me" style="background:#cbd5e1; color:#475569;">Joined</span>` : `<button class="action-btn-small" onclick="app.handleJoinGroup(${gId})">Join</button>`}
                </div>`;
        }).join('');
    }

    // --- Authentication Actions ---
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
        try {
            await this.request('/register', { method: 'POST', body: JSON.stringify({ username, nickname, password }) });
            alert('Registered successfully! Please sign in.');
            this.switchAuthTab('login');
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

    handleLogout() {
        this.stopHeartbeat();
        if (this.ws) this.ws.close();
        this.token = this.user = null;
        localStorage.clear();
        this.showAuth();
    }

    // --- Profile & Group Settings ---
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
        const avatarEl = document.getElementById('my-avatar');
        avatarEl.textContent = (this.user.nickname || 'U')[0].toUpperCase();
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

        // Versioning (Piggybacking) reconciliation
        if (msg.sender_info_version) {
            const cached = this.knownUsers[msg.sender_id];
            if (!cached || msg.sender_info_version > cached.version) {
                const u = await this.request(`/users/${msg.sender_id}`);
                this.knownUsers[u.id] = { nickname: u.nickname, avatar: u.avatar, version: u.info_version };
            }
        }

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
            case 11: 
                alert(`Friend request ${msg.content}`); 
                this.loadInitialData(); // Reload to get new friend info
                break;
            case 12: 
            case 13: 
                if (this.currentChat?.conversation_id === msg.conversation_id) {
                    alert(msg.msg_type === 12 ? 'Removed from group' : 'Group dismissed');
                    this.closeChat();
                }
                this.loadInitialData();
                break;
            case 14:
                // If currently chatting with the person who deleted me
                if (this.currentChat && !this.currentChat.isGroup && this.currentChat.peer_id == msg.sender_id) {
                    alert('The other party has removed you from their friends list.');
                    this.closeChat();
                } else {
                    this.loadInitialData();
                }
                break;
            case 16: alert(`Join request rejected`); this.loadInitialData(); break;
        }
    }

    // --- Action Handlers ---
    async handleApplyFriend(userId) {
        const message = prompt('Intro:', 'Hi, I want to be your friend.');
        if (message !== null) await this.request('/friend/apply', { method: 'POST', body: JSON.stringify({ to_user_id: userId, message }) });
    }

    async handleJoinGroup(groupId) {
        const message = prompt('Intro:');
        if (message !== null) {
            await this.request(`/groups/${groupId}/join`, { method: 'POST', body: JSON.stringify({ message }) });
            alert('Request sent!');
        }
    }

    async handleHandleApply(applyId, accept) {
        await this.request('/friend/apply/handle', { method: 'POST', body: JSON.stringify({ apply_id: applyId, accept }) });
        await this.loadInitialData();
    }

    // --- Group Actions ---
    async handleCreateGroup(name) {
        const g = await this.request('/groups', { method: 'POST', body: JSON.stringify({ name }) });
        await this.loadInitialData(); this.openGroupChat(g.group_id || g.id);
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
                <div class="member-item" onclick="app.submitDirectInvite(${f.user_id})" style="cursor:pointer; border-radius:8px;">
                    <div class="avatar-circle" style="width:32px; height:32px; font-size:11px;">${f.nickname[0].toUpperCase()}</div>
                    <label style="margin-left:12px; cursor:pointer; flex:1; font-weight:600;">${f.nickname}</label>
                    <span style="color:var(--primary); font-size:11px;">Invite</span>
                </div>`).join('');
            document.getElementById('invite-modal').classList.remove('hidden');
        } catch (e) { alert(e.message); }
    }

    async submitDirectInvite(uid) {
        try {
            await this.request(`/groups/${this.currentChat.peer_id}/invite`, { method: 'POST', body: JSON.stringify({ member_ids: [uid] }) });
            document.getElementById('invite-modal').classList.add('hidden');
            alert('Invited!');
        } catch(e) { alert(e.message); }
    }

    async toggleMembers() {
        const panel = document.getElementById('member-list-panel');
        if (!panel.classList.contains('hidden')) return panel.classList.add('hidden');
        try {
            const data = await this.request(`/groups/${this.currentChat.peer_id}/members`);
            const group = this.groups.find(g => (g.id == this.currentChat.peer_id || g.group_id == this.currentChat.peer_id));
            const isOwner = group?.owner_id == this.user.id;
            panel.innerHTML = data.members.map(m => {
                const name = m.nickname || `User ${m.user_id}`;
                const isMe = m.user_id == this.user.id;
                return `
                <div class="member-item">
                    <div class="avatar-circle" style="width:36px; height:36px; font-size:13px; background: #e2e8f0;">${name[0].toUpperCase()}</div>
                    <div class="member-info-text">
                        <div class="member-name-row">
                            <span class="member-display-name">${name}</span>
                            ${isMe ? '<span class="member-role-tag role-me" onclick="app.showMemberSettings()" style="cursor:pointer;">You</span>' : ''}
                            ${m.role === 2 || m.user_id == group?.owner_id ? '<span class="member-role-tag role-owner">Owner</span>' : ''}
                        </div>
                        <div style="font-size: 11px; color: var(--text-muted);">ID: ${m.user_id}</div>
                    </div>
                    ${isOwner && !isMe ? `<button class="action-btn-small danger" onclick="app.handleKickMember(${m.user_id})">Kick</button>` : ''}
                </div>`;
            }).join('');
            panel.classList.remove('hidden');
        } catch (e) {}
    }

    async handleKickMember(userId) {
        if (confirm('Kick?')) {
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

    async handleBlockFriend() {
        if (!this.currentChat || this.currentChat.isGroup) return;
        if (confirm('Block this user? You will no longer receive messages from them.')) {
            try {
                await this.request(`/friends/${this.currentChat.peer_id}/block`, { method: 'POST' });
                alert('Blocked successfully');
                this.closeChat();
            } catch (e) { alert(e.message); }
        }
    }

    async handleDeleteFriend() {
        if (!this.currentChat || this.currentChat.isGroup) return;
        if (confirm('Delete this friend? Chat history will be preserved but they will be removed from your list.')) {
            try {
                await this.request(`/friends/${this.currentChat.peer_id}`, { method: 'DELETE' });
                alert('Friend deleted');
                this.closeChat();
            } catch (e) { alert(e.message); }
        }
    }

    closeChat() {
        this.currentChat = null;
        document.getElementById('chat-view').classList.add('hidden');
        document.getElementById('welcome-view').classList.remove('hidden');
        this.loadInitialData();
    }

    async loadRequests() {
        try {
            const data = await this.request('/friend/apply/list');
            this.requests = data.applies || [];
            this.updateBadge();
            if (this.currentView === 'friends') this.renderFriendList();
        } catch (e) {}
    }

    async loadConversations() {
        try {
            const data = await this.request('/conversations');
            this.conversations = data.conversations || [];
            this.renderConversationList();
        } catch (e) {}
    }

    async loadInitialData() {
        this.updateMyProfile();
        const [c, f, g, r, gr] = await Promise.all([this.request('/conversations'), this.request('/friends'), this.request('/groups'), this.request('/friend/apply/list'), this.request('/groups/requests')]);
        this.conversations = c.conversations || []; this.friends = f.friends || []; this.groups = g.groups || []; this.requests = r.applies || []; this.groupRequests = gr.requests || [];
        this.updateBadge(); this.renderCurrentList();
    }

    renderCurrentList() {
        if (this.currentView === 'chats') this.renderConversationList();
        else if (this.currentView === 'friends') this.renderFriendList();
        else if (this.currentView === 'groups') this.renderGroupList();
    }

    renderConversationList() {
        const sorted = [...this.conversations].sort((a, b) => (b.is_top - a.is_top) || (b.last_message_time - a.last_message_time));
        document.getElementById('list-content').innerHTML = sorted.map(c => {
            const isGroup = c.conversation_id.startsWith('group_');
            let displayName = isGroup ? `Group ${c.peer_id}` : `User ${c.peer_id}`;
            let displayAvatar = isGroup ? 'G' : 'U';

            if (isGroup) {
                const g = this.groups.find(g => (g.group_id == c.peer_id || g.id == c.peer_id));
                if (g) displayName = g.name;
            } else {
                const f = this.friends.find(f => f.user_id == c.peer_id);
                if (f) {
                    displayName = f.nickname;
                    displayAvatar = (f.nickname || 'U')[0].toUpperCase();
                }
            }

            return `
            <div class="list-item ${this.currentChat?.conversation_id === c.conversation_id ? 'active' : ''}" 
                 onclick="app.openChat('${c.conversation_id}', ${c.peer_id}, ${isGroup})">
                <div class="avatar-circle">${displayAvatar}</div>
                <div class="list-item-info">
                    <div class="list-item-title"><span class="list-item-name">${displayName}</span><span class="list-item-time">${this.formatTime(c.last_message_time)}</span></div>
                    <div class="list-item-preview">${c.last_message || '...'}</div>
                </div>
                ${c.unread_count > 0 ? `<span class="badge">${c.unread_count}</span>` : ''}
            </div>`;
        }).join('');
    }

    renderFriendList() {
        const pending = this.requests.filter(r => r.status === 0);
        let html = '';
        if (pending.length) {
            html += `<div class="list-section-title">APPLICATIONS</div>`;
            html += pending.map(r => `<div class="list-item">
                <div class="avatar-circle">?</div>
                <div class="list-item-info"><div class="list-item-name">User ${r.from_user_id}</div><div class="list-item-preview">${r.message || 'Hi'}</div></div>
                <div style="display:flex; gap:10px;"><button class="accept-btn" onclick="app.handleHandleApply(${r.id}, true)">✔</button><button class="reject-btn" onclick="app.handleHandleApply(${r.id}, false)">✖</button></div>
            </div>`).join('');
        }
        html += `<div class="list-section-title">MY FRIENDS</div>`;
        html += this.friends.map(f => `<div class="list-item" onclick="app.openPrivateChat(${f.user_id})">
            <div class="avatar-circle">${(f.nickname || 'U')[0].toUpperCase()}</div>
            <div class="list-item-info"><div class="list-item-name">${f.nickname}</div><div class="list-item-preview">ID: ${f.user_id}</div></div>
            <button class="action-btn-small" onclick="event.stopPropagation(); app.openPrivateChat(${f.user_id})">Chat</button>
        </div>`).join('');
        document.getElementById('list-content').innerHTML = html;
    }

    renderGroupList() {
        const container = document.getElementById('list-content');
        let html = '';
        if (this.groupRequests.length > 0) {
            html += `<div class="list-section-title">GROUP REQUESTS</div>`;
            html += this.groupRequests.map(r => `<div class="list-item">
                <div class="avatar-circle">?</div>
                <div class="list-item-info">
                    <div class="list-item-name">${r.nickname || 'User '+r.user_id} <span style="font-size:10px; opacity:0.6;">-> Group ${r.group_id}</span></div>
                    <div class="list-item-preview">${r.message || 'Wants to join'}</div>
                </div>
                <div style="display:flex; gap:10px;"><button class="accept-btn" onclick="app.handleHandleGroupRequest(${r.id}, true)">✔</button><button class="reject-btn" onclick="app.handleHandleGroupRequest(${r.id}, false)">✖</button></div>
            </div>`).join('');
        }
        html += `<div class="list-section-title">MY GROUPS</div>`;
        html += this.groups.map(g => `<div class="list-item" onclick="app.openGroupChat(${g.group_id || g.id})">
            <div class="avatar-circle">G</div>
            <div class="list-item-info"><div class="list-item-name">${g.name}</div><div class="list-item-preview">Group ID: ${g.group_id || g.id}</div></div>
            <button class="action-btn-small" onclick="event.stopPropagation(); app.openGroupChat(${g.group_id || g.id})">Chat</button>
        </div>`).join('');
        container.innerHTML = html;
    }

    async handleHandleGroupRequest(requestId, accept) {
        await this.request('/groups/requests/handle', { method: 'POST', body: JSON.stringify({ request_id: requestId, accept }) });
        await this.loadInitialData();
    }

    renderMessages() {
        document.getElementById('message-list').innerHTML = this.messages.map(m => {
            // System message check: type 6 OR sender_id 0
            if (m.msg_type === 6 || m.sender_id == 0) {
                return `<div class="message-system"><i class="fas fa-info-circle"></i> ${m.content}</div>`;
            }

            let senderName = 'User ' + m.sender_id;
            if (m.sender_id == this.user.id) {
                senderName = 'You';
            } else {
                // Try cache first
                if (this.knownUsers[m.sender_id]) {
                    senderName = this.knownUsers[m.sender_id].nickname;
                } else {
                    // Try friends list
                    const friend = this.friends.find(f => f.user_id == m.sender_id);
                    if (friend) {
                        senderName = friend.nickname;
                    }
                }
            }

            return `
            <div class="message-row ${m.sender_id == this.user.id ? 'self' : ''}">
                <div class="message-meta">${senderName}, ${this.formatTime(m.timestamp / 1000)}</div>
                <div class="message-bubble ${m.isOptimistic ? 'optimistic' : ''}">${m.content}</div>
            </div>`;
        }).join('');
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
        const pidInt = parseInt(pId);
        this.currentChat = { conversation_id: id, peer_id: pidInt, isGroup: isG };
        document.getElementById('welcome-view').classList.add('hidden');
        document.getElementById('chat-view').classList.remove('hidden');
        document.getElementById('member-list-panel').classList.add('hidden');
        
        let title = isG ? `Group: ${pidInt}` : `User: ${pidInt}`;
        let avatarText = isG ? 'G' : 'U';

        if (isG) {
            const g = this.groups.find(g => (g.group_id == pidInt || g.id == pidInt));
            if (g) title = g.name;
        } else {
            const f = this.friends.find(f => f.user_id == pidInt);
            if (f) {
                title = f.nickname;
                avatarText = (f.nickname || 'U')[0].toUpperCase();
            }
        }
        
        document.getElementById('active-chat-name').textContent = title;
        document.getElementById('active-avatar').textContent = avatarText;
        
        const groupActions = document.getElementById('group-actions');
        const friendActions = document.getElementById('friend-actions');

        if (isG) {
            groupActions.classList.remove('hidden');
            friendActions.classList.add('hidden');
            const group = this.groups.find(g => (g.group_id == pidInt || g.id == pidInt));
            const isOwner = group?.owner_id == this.user.id;
            
            document.getElementById('invite-btn').classList.remove('hidden'); 
            document.getElementById('dismiss-group-btn').classList.toggle('hidden', !isOwner);
            document.getElementById('quit-group-btn').classList.toggle('hidden', isOwner);
            document.getElementById('active-chat-name').onclick = isOwner ? () => this.handleUpdateAnnouncement() : null;
            document.getElementById('active-chat-name').style.cursor = isOwner ? 'pointer' : 'default';
        } else {
            groupActions.classList.add('hidden');
            friendActions.classList.remove('hidden');
        }
        
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
    updateBadge() {
        const badge = document.getElementById('request-badge');
        const count = this.requests.filter(r => r.status === 0).length + this.groupRequests.length;
        badge.textContent = count;
        badge.classList.toggle('hidden', count === 0);
    }
    formatTime(ts) { if (!ts) return ''; const d = new Date(ts * 1000); return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }); }
    scrollToBottom() {
        const el = document.getElementById('message-list');
        if (el) setTimeout(() => { el.scrollTop = el.scrollHeight; }, 50);
    }
}

const app = new GoChatApp();
