# Docka - Tactical Turn-Based RPG

Docka is a data-driven, tactical turn-based RPG featuring a dynamic Combat Readiness (CR) system. The game is built using a modern Client-Server architecture where a highly performant **Go backend** acts as the Authoritative Server, and a **Godot 4 frontend** serves as a purely reactive visual renderer.

## Architecture Overview

*   **Authoritative Backend (docka-backend):** Built in Go. 100% of the game logic, math, stats calculations, Combat Readiness ticks, status decay, effectiveness checks, and combat queues are processed here. The client only sends intents (e.g., "Use Skill 2 on Target A").
*   **Reactive Client (docka-client):** Built in Godot 4 (using the Compatibility renderer). It acts purely as a visual renderer. It receives a JSON payload of the current state and a sequence of visual steps (damage, heals, effects), and plays them using asynchronous queues.

## Current Features

### Backend (Game Logic)
*   **Dynamic CR Tick Engine:** Turn order is dictated by a continuously ticking Combat Readiness gauge. Faster units act more frequently. Includes overflow handling (e.g., hitting 150% CR gives you a head start for the next turn).
*   **Event-Driven Combat Loop:** Features a strict macro combat stack chain (Main Hit -> Counters -> Dual Attacks -> Extra Damage).
*   **Advanced Skill Targeting:** Skills can have multiple effects with distinct targets (e.g., damage a single enemy, but apply an AoE debuff to all enemies).
*   **Complex Skill Mechanics:** 
    *   **Skill Counters:** Persistent stacking (e.g., damage increases by 20% each time a skill is used, up to 5 times).
    *   **Conditional Modifiers:** Defense penetration (e.g., ignore 70% DEF if the target has a shield) and HP-scaling damage.
*   **Status Effects:** Buffs, Debuffs, Shields, Damage over Time (DoT), Immortality (cannot drop below 1 HP), Revive, and "Extinct" states (cannot be revived).
*   **Reactions & Triggers:** Passive counter-attacks, ally counters, and global Extra Damage pooling (AoE explosions that trigger after all main attacks resolve).

### Frontend (User Interface)
*   **Team Selection:** Pick 3 heroes from a roster to build your squad before battle.
*   **Dynamic Character Cards:** Automatically maps selected heroes to the UI, updates names, health bars, and skills.
*   **Health & Shield Bars:** Visual representation of HP and temporary absorption shields.
*   **Status Icon Flow Layout:** Status effects are displayed as colored icons above the character. The UI automatically wraps icons to a new line when space runs out.
*   **Predictive Timeline:** Displays the upcoming turn order showing the *future* Combat Readiness percentage (e.g., 130%) so players can strategize.
*   **Batched Visual Feedback:** Floating text for damage, heals, and buffs. Synchronized effects (like AoE damage) appear simultaneously to feel impactful.

## How to Run and Test

### Prerequisites
*   [Go](https://go.dev/dl/) (Version 1.20 or higher)
*   [Godot Engine](https://godotengine.org/download) (Version 4.x)

### Step 1: Start the Backend Server
1. Open a terminal (Command Prompt, PowerShell, or VS Code terminal).
2. Navigate to the backend folder:
    
    cd docka-backend
    
3. Run the Go server:
    
    go run ./cmd/server
    
4. You should see the message: `Go Server running on http://localhost:8080`.
5. Keep this terminal running in the background.

### Step 2: Start the Frontend Client
1. Open the **Godot Engine**.
2. Click **Import** and select the `project.godot` file inside the `docka-client` folder.
3. Once the editor loads, press **F5** (or click the Play button in the top right corner).

### Step 3: Play the Game
1. **Team Select:** You will be greeted by the Team Selection screen. Click on 3 heroes to add them to your squad (e.g., Knight, Cleric, Anchor Knight). Click **Start Battle**.
2. **Combat:** The battle will begin based on the Combat Readiness timeline. 
3. **Take Action:** When it is a player's turn (indicated by the Gold highlight and the "Your Turn" label), click a skill button.
    *   Skills that require a single target will prompt you to click an enemy or ally card.
    *   Skills that are Self or AoE will execute immediately.
4. **Watch the Results:** Observe the floating damage numbers, the shifting timeline, and the status icons updating in real-time. The battle log at the bottom will explain everything happening under the hood.

## Repository Structure

    docka-game/
    ├── docka-backend/        # Go Authoritative Server
    │   ├── cmd/server/       # Main HTTP server entry point
    │   ├── internal/combat/  # CR engine, turn manager, timeline simulation
    │   └── internal/entity/  # Domain models (Stats, StatusEffects, Entities)
    └── docka-client/         # Godot 4 Reactive Client
        ├── globals/          # Autoloads (ServerAPI, GameState)
        ├── scenes/           # UI Scenes (Battle, TeamSelect)
        └── scenes/entities/  # Reusable UI components (CharacterCard, HealthBar)

## Development Notes
*   The backend uses `sync.Mutex` to prevent data races on the global `CombatState` during HTTP requests.
*   The client relies heavily on `find_child()` to decouple script logic from strict node paths, allowing for easier UI refactoring in the Godot editor.
*   The frontend uses a batching system for visual steps to ensure that simultaneous events (like an AoE explosion hitting all enemies) appear at the exact same frame.