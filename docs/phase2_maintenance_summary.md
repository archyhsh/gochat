# GoChat v2.0.0 Maintenance: System Messaging & Cold Start Fixes

This document details the critical fixes implemented to resolve conversation visibility issues and system messaging robustness.

## 1. System Messaging Strategy (Identity 0)
- **Non-Participant Senders**: The system now supports a `SenderId: 0` for automated notifications (e.g., "User A created the group").
- **Exclusion from Bookmarks**: System senders no longer generate their own `user_conversation` entries, preventing database pollution with "User 0" records.
- **Visual Distinction**: The frontend now renders messages from `sender_id: 0` with a unique italic style and orange border for high visibility.

## 2. Conversation Initialization (Cold Start)
- **Atomic Upsert**: The `ConversationModel.UpdateSeq` method was refactored to use `INSERT ... ON DUPLICATE KEY UPDATE`. This ensures that the global conversation record is created upon the very first message if it doesn't exist.
- **Targeted Visibility (`target_ids`)**: Added a `target_ids` field to `ChatMessageEvent`.
    - **Group Events**: When a group is created, the owner is added to `target_ids`. When a user joins, they are added.
    - **Friend Events**: When a friend request is accepted, both users are added to `target_ids` for their respective reciprocal notifications.
    - **Logic**: `SaveMessageLogic` ensures that any user in `target_ids` has their local `user_conversation` record initialized, making the chat immediately visible in their list.
- **GetMessages Fallback**: `GetMessagesLogic` now falls back to checking the global `conversation` table if a user's local bookmark is missing. This prevents 404 errors when a user opens a legitimate conversation for the first time.

## 3. Stability & Compatibility
- **Safe Type Conversion**: Implemented a `toInt64` helper in the Kafka consumer to prevent panics when decoding JSON numbers from various upstream services (handling `nil`, `float64`, and `string` types).
- **Cleanup**: Performed a systematic cleanup of the `rpc/message` directory, removing redundant underscore-style files that caused redeclaration errors.
- **Unified Front-end**: Merged friend requests into the "Friends" view and implemented professional member management (Invite/Kick) within the Group view.
