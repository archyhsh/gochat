// Constants for API endpoints
const API_BASE = ''; // The backend currently doesn't use the /api prefix

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
        this.knownUsers = {}; // Cache for userId -> userInfo
        this.currentView = 'chats';
        this.searchTimer = null;
        
        this.init();
    }

    init() {
        this.bindEvents();
        if (this.token) {
            this.showApp();
            this.loadInitialData();
        } else {
            this.showAuth();
        }
    }

    bindEvents() {
        // Tab switching
        document.querySelectorAll('.auth-tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                const type = e.target.dataset.type;
                this.switchAuthTab(type);
            });
        });

        // Login form
        document.getElementById('login-btn')?.addEventListener('click', () => this.handleLogin());
        document.getElementById('register-btn')?.addEventListener('click', () => this.handleRegister());
        
        // Logout
        document.getElementById('logout-btn')?.addEventListener('click', () => this.handleLogout());

        // Sidebar navigation
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', (e) => {
                const view = e.currentTarget.dataset.view;
                this.switchSidebarView(view);
            });
        });

        // Search logic
        const searchInput = document.getElementById('global-search');
        searchInput?.addEventListener('input', (e) => this.onSearchInput(e));

        // Create group
        document.getElementById('create-group-btn')?.addEventListener('click', () => this.handleCreateGroup());

        // Chat input
        document.getElementById('chat-input')?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this.handleSendMessage();
        });
        document.getElementById('send-msg-btn')?.addEventListener('click', () => this.handleSendMessage());

        // Group actions
        document.getElementById('members-btn')?.addEventListener('click', () => this.toggleMemberList());
        document.getElementById('invite-btn')?.addEventListener('click', () => this.handleInviteMember());
    }

    // --- UI State Management ---

    showAuth() {
        document.getElementById('auth-page').classList.remove('hidden');
        document.getElementById('app-page').classList.add('hidden');
    }

    showApp() {
        document.getElementById('auth-page').classList.add('hidden');
        document.getElementById('app-page').classList.remove('hidden');
        if (this.user) {
            document.getElementById('my-name').textContent = this.user.nickname;
            document.getElementById('my-avatar').textContent = (this.user.nickname || 'U').charAt(0).toUpperCase();
        }
    }

    switchAuthTab(type) {
        document.querySelectorAll('.auth-tab').forEach(t => t.classList.toggle('active', t.dataset.type === type));
        document.getElementById('login-form').classList.toggle('hidden', type !== 'login');
        document.getElementById('register-form').classList.toggle('hidden', type !== 'register');
    }

    switchSidebarView(view) {
        this.currentView = view;
        document.querySelectorAll('.nav-item').forEach(i => i.classList.toggle('active', i.dataset.view === view));
        
        // Update Search Placeholder based on view
        const searchInput = document.getElementById('global-search');
        if (searchInput) {
            searchInput.value = '';
            if (view === 'chats') {
                searchInput.placeholder = 'Search existing friends and groups...';
            } else if (view === 'friends') {
                searchInput.placeholder = "Search new friends' nickname...";
            } else if (view === 'groups') {
                searchInput.placeholder = "Search new groups' name...";
            } else {
                searchInput.placeholder = "Search...";
            }
        }

        if (view === 'chats') this.loadConversations();
        if (view === 'friends') {
            this.loadFriends();
            this.loadApplyList();
        }
        if (view === 'groups') this.loadGroups();
    }

    // --- API Interactions ---

    async request(path, options = {}) {
        const url = `${API_BASE}${path}`;
        const token = options.token || this.token;
        const headers = {
            'Content-Type': 'application/json',
            ...(token ? { 'Authorization': `Bearer ${token}` } : {}),
            ...options.headers
        };

        try {
            const response = await fetch(url, { ...options, headers });
            if (response.status === 204) return null;
            const data = await response.json();
            
            if (!response.ok) {
                if (response.status === 401) {
                    console.warn('Unauthorized request, logging out...');
                    this.handleLogout();
                }
                throw new Error(data.message || data.desc || 'Request failed');
            }
            return data;
        } catch (err) {
            console.error(`API Error (${path}):`, err);
            throw err;
        }
    }

    async handleLogin() {
        const username = document.getElementById('login-username').value;
        const password = document.getElementById('login-password').value;
        const errorEl = document.getElementById('auth-error');

        try {
            const data = await this.request('/login', {
                method: 'POST',
                body: JSON.stringify({ username, password })
            });

            // 1. Update and persist token immediately
            this.token = data.token;
            localStorage.setItem('token', this.token);
            
            // 2. Fetch profile using the fresh token
            const me = await this.request('/user/me', { token: this.token });
            
            // 3. Update user data
            this.user = me;
            localStorage.setItem('user', JSON.stringify(this.user));
            
            // 4. Update UI and load data
            this.showApp();
            this.loadInitialData();
        } catch (err) {
            errorEl.textContent = err.message;
            errorEl.classList.remove('hidden');
        }
    }

    async handleRegister() {
        const username = document.getElementById('reg-username').value;
        const nickname = document.getElementById('reg-nickname').value;
        const password = document.getElementById('reg-password').value;
        const errorEl = document.getElementById('auth-error');

        try {
            await this.request('/register', {
                method: 'POST',
                body: JSON.stringify({ username, nickname, password })
            });
            this.switchAuthTab('login');
            alert('Registration successful! Please login.');
        } catch (err) {
            errorEl.textContent = err.message;
            errorEl.classList.remove('hidden');
        }
    }

    handleLogout() {
        this.token = null;
        this.user = null;
        localStorage.clear();
        this.showAuth();
    }

    // --- Search Logic ---

    onSearchInput(e) {
        clearTimeout(this.searchTimer);
        const keyword = e.target.value.trim().toLowerCase();
        
        if (!keyword) {
            this.renderActiveView();
            return;
        }

        this.searchTimer = setTimeout(async () => {
            if (this.currentView === 'chats') {
                this.searchLocalConversations(keyword);
            } else if (this.currentView === 'friends') {
                this.searchRemoteUsers(keyword);
            } else if (this.currentView === 'groups') {
                this.searchRemoteGroups(keyword);
            }
        }, 500);
    }

    searchLocalConversations(keyword) {
        const filteredFriends = this.friends.filter(f => 
            (f.nickname && f.nickname.toLowerCase().includes(keyword)) || 
            (f.remark && f.remark.toLowerCase().includes(keyword))
        );
        const filteredGroups = this.groups.filter(g => 
            g.name && g.name.toLowerCase().includes(keyword)
        );
        
        const container = document.getElementById('list-content');
        container.innerHTML = '';

        if (filteredFriends.length === 0 && filteredGroups.length === 0) {
            container.innerHTML = '<div class="empty-state"><p>No friends or groups found</p></div>';
            return;
        }

        filteredFriends.forEach(f => {
            const el = document.createElement('div');
            el.className = 'list-item';
            el.innerHTML = `
                <div class="avatar-circle">${f.nickname.charAt(0).toUpperCase()}</div>
                <div class="list-item-info">
                    <div class="list-item-title">
                        <span class="list-item-name">${f.remark || f.nickname}</span>
                        <button class="action-btn-small" onclick="event.stopPropagation(); window.app.startPrivateChatById(${f.user_id}, '${f.remark || f.nickname}')">Chat</button>
                    </div>
                    <div class="list-item-preview">Friend</div>
                </div>
            `;
            container.appendChild(el);
        });

        filteredGroups.forEach(g => {
            const el = document.createElement('div');
            el.className = 'list-item';
            el.innerHTML = `
                <div class="avatar-circle">G</div>
                <div class="list-item-info">
                    <div class="list-item-title">
                        <span class="list-item-name">${g.name}</span>
                        <button class="action-btn-small" onclick="event.stopPropagation(); window.app.startGroupChatById(id, '${g.name}')">Chat</button>
                    </div>
                    <div class="list-item-preview">Group</div>
                </div>
            `;
            container.appendChild(el);
        });
    }

    async searchRemoteUsers(keyword) {
        try {
            const data = await this.request(`/users/search?keyword=${encodeURIComponent(keyword)}`);
            this.renderSearchResults(data.users || [], 'users');
        } catch (err) {
            console.error('Search failed', err);
        }
    }

    async searchRemoteGroups(keyword) {
        try {
            const data = await this.request(`/groups/search?keyword=${encodeURIComponent(keyword)}`);
            this.renderSearchResults(data.groups || [], 'groups');
        } catch (err) {
            console.error('Search failed', err);
        }
    }

    renderSearchResults(results, type) {
        const container = document.getElementById('list-content');
        container.innerHTML = '';
        
        if (results.length === 0) {
            container.innerHTML = '<div class="empty-state"><p>No results found</p></div>';
            return;
        }

        results.forEach(item => {
            const el = document.createElement('div');
            el.className = 'list-item';
            
            if (type === 'users') {
                if (item.id === this.user.id) return; // Skip self
                const isFriend = this.friends.some(f => f.user_id === item.id);
                el.innerHTML = `
                    <div class="avatar-circle">${item.nickname.charAt(0).toUpperCase()}</div>
                    <div class="list-item-info">
                        <div class="list-item-title">
                            <span class="list-item-name">${item.nickname}</span>
                            ${isFriend ? 
                                `<span class="list-item-time">Friend</span>` : 
                                `<button class="prominent-add-btn" onclick="event.stopPropagation(); window.app.handleApplyFriendPrompt(${item.id})">Add</button>`
                            }
                        </div>
                        <div class="list-item-preview">@${item.username}</div>
                    </div>
                `;
            } else {
                const isMember = this.groups.some(g => g.id === item.id);
                el.innerHTML = `
                    <div class="avatar-circle">G</div>
                    <div class="list-item-info">
                        <div class="list-item-title">
                            <span class="list-item-name">${item.name}</span>
                            ${isMember ? 
                                `<span class="list-item-time">Member</span>` : 
                                `<button class="prominent-add-btn" onclick="event.stopPropagation(); window.app.joinGroup(${item.id})">Join</button>`
                            }
                        </div>
                        <div class="list-item-preview">${item.description || 'Public Group'}</div>
                    </div>
                `;
            }
            container.appendChild(el);
        });
    }

    // --- Social Actions ---

    async handleApplyFriendPrompt(targetId) {
        const message = prompt("Enter invitation message:", "Hello, I want to add you as a friend.");
        if (message === null) return;
        this.applyFriend(targetId, message);
    }

    async applyFriend(targetId, message) {
        try {
            await this.request('/friend/apply', {
                method: 'POST',
                body: JSON.stringify({ to_user_id: targetId, message: message })
            });
            alert('Friend request sent!');
        } catch (err) {
            alert('Failed to send request: ' + err.message);
        }
    }

    async joinGroup(groupId) {
        const intro = prompt("Please enter your introduction:", "I'm " + (this.user?.nickname || 'a new member'));
        if (intro === null) return;

        try {
            await this.request(`/groups/${groupId}/join`, {
                method: 'POST',
                body: JSON.stringify({ message: intro })
            });
            alert('Successfully joined group!');
            await this.loadGroups();
            this.switchSidebarView('groups');
        } catch (err) {
            alert('Failed to join group: ' + err.message);
        }
    }

    async handleCreateGroup() {
        const name = prompt("Enter group name:");
        if (!name) return;
        const description = prompt("Enter group description (optional):", "");

        try {
            const data = await this.request('/groups', {
                method: 'POST',
                body: JSON.stringify({ name, description, avatar: '' })
            });
            alert(`Group "${data.name}" created!`);
            await this.loadGroups();
            this.switchSidebarView('groups');
        } catch (err) {
            alert('Failed to create group: ' + err.message);
        }
    }

    async loadApplyList() {
        try {
            const data = await this.request('/friend/apply/list');
            this.requests = data.applies || [];
            this.updateRequestBadge();
            
            // Try to resolve nicknames for the requesters
            for (let req of this.requests) {
                if (!this.knownUsers[req.from_user_id]) {
                    this.resolveUserInfo(req.from_user_id);
                }
            }

            if (this.currentView === 'friends') {
                this.renderFriendList();
            }
        } catch (err) {
            console.error('Failed to load apply list', err);
        }
    }

    async resolveUserInfo(userId) {
        try {
            const user = await this.request(`/users/${userId}`);
            if (user) {
                this.knownUsers[userId] = user;
                if (this.currentView === 'friends') this.renderFriendList();
                if (this.currentChat) this.renderMessages();
            }
        } catch (err) {
            console.warn(`Could not resolve info for user ${userId}`, err);
        }
    }

    updateRequestBadge() {
        const badge = document.getElementById('request-badge');
        const count = this.requests.filter(r => r.status === 0).length;
        if (count > 0) {
            badge.textContent = count;
            badge.classList.remove('hidden');
        } else {
            badge.classList.add('hidden');
        }
    }

    async handleApply(applyId, accept) {
        try {
            await this.request('/friend/apply/handle', {
                method: 'POST',
                body: JSON.stringify({ apply_id: applyId, accept: accept })
            });
            alert(accept ? 'Accepted!' : 'Rejected!');
            await this.loadApplyList();
            if (accept) await this.loadFriends();
        } catch (err) {
            alert('Operation failed: ' + err.message);
        }
    }

    // --- Member Management ---

    async toggleMemberList() {
        const panel = document.getElementById('member-list-panel');
        if (!panel.classList.contains('hidden')) {
            panel.classList.add('hidden');
            return;
        }

        if (!this.currentChat || !this.currentChat.conversation_id.startsWith('group_')) return;
        const groupId = this.currentChat.peer_id;

        try {
            const data = await this.request(`/groups/${groupId}/members`);
            const members = data.members || [];
            
            const group = this.groups.find(g => g.id === groupId);
            const isOwner = group && group.owner_id === this.user.id;

            panel.innerHTML = '<div style="padding: 10px; font-weight: bold; font-size: 12px; color: var(--text-muted);">MEMBERS</div>';
            members.forEach(m => {
                const el = document.createElement('div');
                el.className = 'list-item';
                el.style = "padding: 8px 20px;";
                const roleText = m.role === 2 ? 'Owner' : (m.role === 1 ? 'Admin' : '');
                
                el.innerHTML = `
                    <div class="avatar-circle" style="width: 24px; height: 24px; font-size: 10px;">${m.nickname.charAt(0).toUpperCase()}</div>
                    <div class="list-item-info">
                        <div class="list-item-title">
                            <span class="list-item-name" style="font-size: 13px;">${m.nickname} ${roleText ? `<span style="color: var(--warning-color); font-size: 10px;">(${roleText})</span>` : ''}</span>
                            ${isOwner && m.user_id !== this.user.id ? `
                                <button class="reject-btn" style="padding: 2px 6px; font-size: 10px;" onclick="event.stopPropagation(); window.app.handleKick(${groupId}, ${m.user_id})">Kick</button>
                            ` : ''}
                        </div>
                    </div>
                `;
                panel.appendChild(el);
            });
            panel.classList.remove('hidden');
        } catch (err) {
            alert('Failed to load members: ' + err.message);
        }
    }

    async handleKick(groupId, userId) {
        if (!confirm('Are you sure you want to kick this member?')) return;
        try {
            await this.request(`/groups/${groupId}/kick/${userId}`, { method: 'POST' });
            alert('Member kicked');
            this.toggleMemberList(); // Refresh panel
        } catch (err) {
            alert('Failed to kick: ' + err.message);
        }
    }

    async handleInviteMember() {
        if (!this.currentChat || !this.currentChat.conversation_id.startsWith('group_')) return;
        const groupId = this.currentChat.peer_id;
        
        // Show a list of friends to invite
        const container = document.getElementById('list-content');
        container.innerHTML = '<div style="padding: 16px; font-weight: bold;">Select friend to invite:</div>';
        
        const friendsToInvite = this.friends; // In real app, filter those already in group
        
        if (friendsToInvite.length === 0) {
            container.innerHTML += '<div class="empty-state"><p>No friends available to invite</p></div>';
            return;
        }

        friendsToInvite.forEach(f => {
            const el = document.createElement('div');
            el.className = 'list-item';
            el.innerHTML = `
                <div class="avatar-circle">${f.nickname.charAt(0).toUpperCase()}</div>
                <div class="list-item-info">
                    <div class="list-item-title">
                        <span class="list-item-name">${f.remark || f.nickname}</span>
                        <button class="prominent-add-btn" onclick="event.stopPropagation(); window.app.executeInvite(${groupId}, ${f.user_id})">Invite</button>
                    </div>
                </div>
            `;
            container.appendChild(el);
        });
    }

    async executeInvite(groupId, userId) {
        try {
            await this.request(`/groups/${groupId}/invite`, {
                method: 'POST',
                body: JSON.stringify({ member_ids: [userId] })
            });
            alert('Invitation sent!');
            this.loadInitialData();
            this.switchSidebarView('chats');
        } catch (err) {
            alert('Failed to invite: ' + err.message);
        }
    }

    // --- Conversation Management ---

    renderActiveView() {
        if (this.currentView === 'chats') this.renderConversationList();
        if (this.currentView === 'friends') this.renderFriendList();
        if (this.currentView === 'groups') this.renderGroupList();
    }

    async startPrivateChatById(userId, name) {
        const convId = `conv_${Math.min(this.user.id, userId)}_${Math.max(this.user.id, userId)}`;
        this.ensureAndSelectConversation(convId, userId, name);
    }

    async startGroupChatById(groupId, name) {
        const convId = `group_${groupId}`;
        this.ensureAndSelectConversation(convId, groupId, name);
    }

    ensureAndSelectConversation(convId, peerId, name) {
        let conv = this.conversations.find(c => c.conversation_id === convId);
        if (!conv) {
            conv = {
                conversation_id: convId,
                peer_id: peerId,
                unread_count: 0,
                last_message: 'Start of a new conversation',
                last_message_time: Math.floor(Date.now() / 1000)
            };
            this.conversations.unshift(conv);
        }
        this.switchSidebarView('chats');
        this.selectConversation(conv, name);
    }

    // --- Data Loading ---

    async loadInitialData() {
        await Promise.all([this.loadFriends(), this.loadGroups(), this.loadApplyList()]);
        this.loadConversations();
    }

    async loadConversations() {
        try {
            const data = await this.request('/conversations');
            this.conversations = data.conversations || [];
            this.renderConversationList();
        } catch (err) {
            console.error('Failed to load conversations', err);
        }
    }

    async loadFriends() {
        try {
            const data = await this.request('/friends');
            this.friends = data.friends || [];
            if (this.currentView === 'friends') this.renderFriendList();
        } catch (err) {
            console.error('Failed to load friends', err);
        }
    }

    async loadGroups() {
        try {
            const data = await this.request('/groups');
            this.groups = data.groups || [];
            if (this.currentView === 'groups') this.renderGroupList();
        } catch (err) {
            console.error('Failed to load groups', err);
        }
    }

    // --- Rendering Helpers ---

    getDisplayName(conv) {
        if (conv.conversation_id === 'system') return 'System Notification';
        if (conv.conversation_id.startsWith('group_')) {
            const g = this.groups.find(x => x.id === conv.peer_id);
            return g ? g.name : `Group ${conv.peer_id}`;
        } else {
            if (conv.peer_id === 0) return 'System';
            const f = this.friends.find(x => x.user_id === conv.peer_id);
            return f ? (f.remark || f.nickname) : (this.knownUsers[conv.peer_id]?.nickname || `User ${conv.peer_id}`);
        }
    }

    renderConversationList() {
        const container = document.getElementById('list-content');
        container.innerHTML = '';
        
        if (this.conversations.length === 0) {
            container.innerHTML = '<div class="empty-state"><p>No conversations</p></div>';
            return;
        }

        this.conversations.forEach(conv => {
            const name = this.getDisplayName(conv);
            const item = document.createElement('div');
            item.className = `list-item ${this.currentChat?.conversation_id === conv.conversation_id ? 'active' : ''}`;
            const avatarChar = conv.peer_id === 0 ? 'S' : name.charAt(0).toUpperCase();
            
            item.innerHTML = `
                <div class="avatar-circle" style="${conv.peer_id === 0 ? 'background: var(--warning-color)' : ''}">${avatarChar}</div>
                <div class="list-item-info">
                    <div class="list-item-title">
                        <span class="list-item-name">${name}</span>
                        <span class="list-item-time">${this.formatTime(conv.last_message_time)}</span>
                    </div>
                    <div class="list-item-preview">${conv.last_message || 'No messages yet'}</div>
                </div>
            `;
            item.onclick = () => this.selectConversation(conv, name);
            container.appendChild(item);
        });
    }

    renderFriendList() {
        const container = document.getElementById('list-content');
        container.innerHTML = '';
        
        // 1. Render Pending Requests first
        const pending = this.requests.filter(r => r.status === 0);
        if (pending.length > 0) {
            const header = document.createElement('div');
            header.style = "padding: 8px 16px; font-size: 11px; color: var(--text-muted); font-weight: bold; text-transform: uppercase;";
            header.textContent = "Friend Requests";
            container.appendChild(header);

            pending.forEach(req => {
                const el = document.createElement('div');
                el.className = 'list-item';
                const senderName = this.knownUsers[req.from_user_id]?.nickname || `User ${req.from_user_id}`;
                el.innerHTML = `
                    <div class="avatar-circle" style="background: var(--warning-color)">${senderName.charAt(0).toUpperCase()}</div>
                    <div class="list-item-info">
                        <div class="list-item-title">
                            <span class="list-item-name">${senderName}</span>
                            <div style="display: flex; gap: 4px;">
                                <button class="accept-btn" onclick="event.stopPropagation(); window.app.handleApply(${req.id}, true)">✔</button>
                                <button class="reject-btn" onclick="event.stopPropagation(); window.app.handleApply(${req.id}, false)">✖</button>
                            </div>
                        </div>
                        <div class="list-item-preview">${req.message || 'Wants to be your friend'}</div>
                    </div>
                `;
                container.appendChild(el);
            });

            const divider = document.createElement('div');
            divider.style = "height: 1px; background: var(--border-color); margin: 8px 16px;";
            container.appendChild(divider);
        }

        // 2. Render Existing Friends
        if (this.friends.length === 0) {
            if (pending.length === 0) {
                container.innerHTML = '<div class="empty-state"><p>No friends yet</p></div>';
            }
            return;
        }

        this.friends.forEach(friend => {
            const item = document.createElement('div');
            item.className = 'list-item';
            item.innerHTML = `
                <div class="avatar-circle">${friend.nickname.charAt(0).toUpperCase()}</div>
                <div class="list-item-info">
                    <div class="list-item-title">
                        <span class="list-item-name">${friend.remark || friend.nickname}</span>
                        <button class="action-btn-small" onclick="event.stopPropagation(); window.app.startPrivateChatById(${friend.user_id}, '${friend.remark || friend.nickname}')">Chat</button>
                    </div>
                    <div class="list-item-preview">@${friend.user_id}</div>
                </div>
            `;
            container.appendChild(item);
        });
    }

    renderGroupList() {
        const container = document.getElementById('list-content');
        container.innerHTML = '';
        
        if (this.groups.length === 0) {
            container.innerHTML = '<div class="empty-state"><p>No groups yet</p></div>';
            return;
        }

        this.groups.forEach(group => {
            const item = document.createElement('div');
            item.className = 'list-item';
            item.innerHTML = `
                <div class="avatar-circle">G</div>
                <div class="list-item-info">
                    <div class="list-item-title">
                        <span class="list-item-name">${group.name}</span>
                        <button class="action-btn-small" onclick="event.stopPropagation(); window.app.startGroupChatById(${group.id}, '${group.name}')">Chat</button>
                    </div>
                    <div class="list-item-preview">${group.description || ''}</div>
                </div>
            `;
            container.appendChild(item);
        });
    }

    async selectConversation(conv, displayName) {
        this.currentChat = conv;
        document.getElementById('welcome-view').classList.add('hidden');
        document.getElementById('chat-view').classList.remove('hidden');
        document.getElementById('active-chat-name').textContent = displayName;
        
        // Toggle group actions visibility
        const isGroup = conv.conversation_id.startsWith('group_');
        document.getElementById('group-actions').classList.toggle('hidden', !isGroup);
        document.getElementById('member-list-panel').classList.add('hidden');

        this.renderConversationList(); // Update active state
        this.loadMessages(conv.conversation_id);
        
        if (conv.unread_count > 0) {
            try {
                await this.request('/conversations/clear_unread', {
                    method: 'POST',
                    body: JSON.stringify({ conversation_id: conv.conversation_id })
                });
                conv.unread_count = 0;
                this.renderConversationList();
                this.loadApplyList(); // Refresh badge
            } catch (err) {
                console.error('Failed to clear unread', err);
            }
        }
    }

    async loadMessages(convId) {
        try {
            const data = await this.request(`/messages?conversation_id=${convId}&limit=50`);
            this.messages = data.messages || [];
            this.renderMessages();
        } catch (err) {
            console.error('Failed to load messages', err);
        }
    }

    renderMessages() {
        const container = document.getElementById('message-list');
        container.innerHTML = '';
        
        this.messages.forEach(msg => {
            const isSelf = msg.sender_id === this.user.id;
            const isSystem = msg.sender_id === 0;
            const row = document.createElement('div');
            row.className = `message-row ${isSelf ? 'self' : ''}`;
            
            let senderName = isSelf ? 'You' : (isSystem ? 'System' : (this.knownUsers[msg.sender_id]?.nickname || `User ${msg.sender_id}`));
            if (!isSelf && !isSystem) {
                const f = this.friends.find(x => x.user_id === msg.sender_id);
                if (f) senderName = f.remark || f.nickname;
            }

            row.innerHTML = `
                <div class="message-meta">${senderName} • ${this.formatTime(msg.timestamp)}</div>
                <div class="message-bubble" style="${isSystem ? 'background: var(--bg-header); font-style: italic; border-left: 3px solid var(--warning-color)' : ''}">
                    ${msg.content}
                </div>
            `;
            container.appendChild(row);
        });
        
        container.scrollTop = container.scrollHeight;
    }

    handleSendMessage() {
        const input = document.getElementById('chat-input');
        const content = input.value.trim();
        if (!content || !this.currentChat) return;

        console.log('Sending message:', content, 'to', this.currentChat.conversation_id);
        
        const newMsg = {
            sender_id: this.user.id,
            content: content,
            timestamp: Date.now(),
            conversation_id: this.currentChat.conversation_id
        };
        
        this.messages.push(newMsg);
        this.renderMessages();
        input.value = '';
    }

    formatTime(ts) {
        if (!ts || ts === 0) return '';
        const date = new Date(ts > 1e11 ? ts : ts * 1000);
        return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    }
}

// Initialize the app
document.addEventListener('DOMContentLoaded', () => {
    window.app = new GoChatApp();
});
