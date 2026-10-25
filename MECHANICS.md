# Docka - Game Mechanics & Design Document

This document serves as the absolute source of truth for the current state of the project. It defines the architecture, systems, and rules that govern the game.

## 1. Architecture Overview

*   **Authoritative Server (Go Backend):** All math, state mutations, turn order, status decay, RNG (hit/effect/resistance), and combat triggers are processed strictly here. The client only sends intents.
*   **Reactive Client (Godot 4 Frontend):** Acts purely as a visual renderer. It receives a JSON payload with the new state and a sequence of visual steps (damage, heals, buffs) and plays them using an asynchronous queue.
*   **Theme:** A DACOMP (UFRGS Computer Science) tactical RPG where friends, social groups, and campus life (like the RU and the Lojinha) are translated into factions, subclasses, and skills.

## 2. The Combat Readiness (CR) Engine

Instead of fixed turns, the game uses a Dynamic Tick Loop based on speed.

*   **Tick Loop:** At each tick, entities gain CR based on their EffectiveSpeed.
*   **Turn Threshold:** A unit acts when their CR reaches 100% or more.
*   **FIFO Tie-breaker:** If multiple units cross 100% simultaneously, the one who reached it first (lower ReachedCRAtTick) goes first.
*   **CR Overflow:** When a unit acts, 100% is subtracted from their CR. If they had 150%, they act and keep 50% for their next turn.
*   **Predictive Timeline:** The UI displays the predicted CR (e.g., 130%) at the exact moment the unit will act, not their current CR.

## 3. The DACOMP Factions (Elements)

Factions dictate damage multipliers and general kit tendencies. The wheel works in a rock-paper-scissors format.

| Faction | Identity / Tendency | Advantage Against | Disadvantage Against |
| :--- | :--- | :--- | :--- |
| Meta (Sofá) | Strategy, buffs, precision. | Chaos | Physical |
| Chaos (Assemelhados) | Unpredictability, evasion, burst. | Structure | Meta |
| Structure (Bolsistas) | Discipline, high HP, defense, healing. | Toxic | Chaos |
| Toxic (Fumantes) | Attrition, DoTs, debuffs. | Physical | Structure |
| Physical (Sinuca) | Brute force, raw damage, counters. | Meta | Toxic |
| Ruler (Lojinha) | Control, stuns, punishing rule-breakers. | Chaos & Physical | Neutral |

*   **Advantage:** Deals 1.5x damage.
*   **Disadvantage:** Deals 0.75x damage and reduces Hit Chance by 50%.

## 4. Subclasses (Inclinations)

Subclasses do not change the Faction wheel; they dictate the behavior or triggers of the character's kit.

*   **Caffeine:** At the start of the turn, gains +15% Combat Readiness.
*   **Beer:** If HP is > 70% at the start of the turn, gains Joy (+20% Damage).
*   **Smoke:** On a successful Basic Attack (Skill 1), has a 40% chance to apply BLIND (reduces enemy Hit Chance).
*   **Green:** On a successful Basic Attack (Skill 1), has a 50% chance to apply SLOW (reduces Speed).
*   **Truco:** On a successful Basic Attack (Skill 1), has a 30% chance to gain +20% Combat Readiness (bluffing to act faster).

## 5. Hit, Effect, and Resistance (E7 System)

Every attack and debuff follows a strict 3-phase resolution process.

**Phase 1: Hit Chance**
*   Base Hit Chance is 100%.
*   Blind: Reduces Hit Chance by 50%.
*   Elemental Disadvantage: Reduces Hit Chance by 50%.
*   Evasion: Reduces Hit Chance by the target's Evasion stat (max 90%).
*   Result: If the attack misses, it deals no damage and cannot apply any debuffs.

**Phase 2: Effect Chance**
*   If the attack hits, the skill's base EffectChance (e.g., 75%) is rolled. If the roll fails, the debuff is not applied.

**Phase 3: Effectiveness vs Resistance**
*   If Phase 2 passes, the server rolls: EffectiveResistance = Max(15, 100 + TargetRes - AttackerEff).
*   15% Rule: A target always has at least a 15% chance to resist a debuff, no matter how high the attacker's Effectiveness is.

## 6. The Combat Stack Chain (Resolution Order)

When a skill is cast, the following phases happen in strict order:

1.  **Phase 1: Main Hit Instance** (Direct damage, buffs, and debuffs apply here).
2.  **Phase 2: Passive Shields & Defenses** (Shields absorb damage before HP is lost).
3.  **Phase 3: Counter-Attacks** (If a unit survives and has a Counter passive/stance, they strike back).
4.  **Phase 4: Dual Attacks** (If a skill triggers an ally to assist, it resolves here).
5.  **Phase 5: Extra Damage** (Delayed AoE explosions or secondary hits resolve as the absolute last action of the turn).

## 7. Skills & Damage Calculations

*   **Skill 1 (Basic Attack):** Always deals 1.0x damage. Used to trigger Subclass effects and reactions.
*   **Skill 2 (Utility/Medium):** Cooldown-based, applies buffs, debuffs, or dual attacks.
*   **Skill 3 (Ultimate):** High impact, long cooldown, usually AoE or executes.
*   **Lifesteal:** Heals the attacker based on the actual HP lost by the target (calculated before reflect damage).
*   **Defense Penetration:** If a target has a shield, a skill with this tag ignores a percentage of their DEF.
*   **Skill Stacking:** Some skills have MaxStacks and StackMultiplier (e.g., deal +20% damage per stack, up to 5 times).

## 8. Status Effects

*   **Buffs:** HASTE (Speed up), DEF_UP (Defense up), JOY (Damage up).
*   **Debuffs:** POISON (DoT), BLIND (Hit chance down), SLOW (Speed down), DEF_DOWN (Defense down), STUN (Lose turn).
*   **Shield:** A temporary HP buffer that absorbs damage before actual HP.
*   **Immortality:** HP cannot drop below 1. Once the buff expires or is removed, the unit can be killed.
*   **Revive:** Brings a dead unit back to life with a percentage of their Max HP.
*   **Extinct (Unrevivable):** A unit killed by an Execute skill or with an unrevivable flag cannot be revived.
*   **Banish:** Removed from the timeline entirely. CR does not tick, and they cannot be targeted. Returns only when the caster dies or the duration expires.

## 9. Reactions & Triggers

*   **Active Interceptors:** Monitors the event bus. If a specific condition is met (e.g., receiving a debuff), it fires immediately. Example: The ANTIDOTE passive cleanses a poison the exact moment it is applied.
*   **Counters:** Triggered by the Main Hit. Does not trigger off Dual Attacks.
*   **Dual Attacks:** Triggered by the skill itself. Does not trigger Counters.
*   **Global Extra Damage Pool:** All Extra Damage multipliers generated during the turn are accumulated and applied simultaneously to all valid targets at the very end of the phase.

## 10. Visual Rendering Rules (Godot)

*   **Batching:** Consecutive visual steps of the same type (e.g., AoE damage) are grouped into a single batch. Godot spawns all floating text simultaneously to make AoE feel impactful.
*   **Timeline:** Smoothly tweens the labels to their new positions when the turn order changes.
*   **Status Icons:** Uses a FlowContainer to wrap icons to a new line upwards when space runs out.