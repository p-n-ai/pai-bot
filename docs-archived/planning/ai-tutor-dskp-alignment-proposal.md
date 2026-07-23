# Proposal: Aligning AI Tutor Core Prompts to KPM DSKP Pedagogy

## Overview
Recent review of the KPM DSKP framework (Tingkatan 2 Mathematics) revealed several foundational pedagogical philosophies that dictate *how* Mathematics should be taught, beyond just the learning standards. Currently, our AI Tutor system prompt primarily focuses on our proprietary dual-loop solving pattern.

To ensure the `pai-bot` acts as a highly authentic KSSM master tutor, we propose updating the underlying AI chatbot prompt to explicitly mandate these KPM philosophies.

## Proposed System Prompt Injections (For Team Discussion)

### 1. Mathematical Fikrah & 4-Step Problem Solving
The DSKP emphasizes that problem solving must be systematic. The AI should stop using generic scaffolding and instead explicitly guide the student through the KPM's 4-step cycle:
1.  **Understanding and interpreting problems** ("What do we know? What are we trying to find?")
2.  **Devising a strategy** ("How should we set this up?")
3.  **Implementing the strategy** ("Let's calculate step-by-step.")
4.  **Doing reflection** ("Does this answer make sense in the real world? Let's check.")

*Impact on Bot:* The AI must not skip straight to "Let's calculate." It must enforce Step 1 and Step 2 first.

### 2. Communication in Mathematics & Reasoning
The DSKP explicitly states that students must "justify their views" and give a "logical explanation."
*Impact on Bot:* The AI must be constrained against accepting "naked numbers." If a student replies with just "42", the AI should respond: *"Correct, but in line with KSSM, tell me exactly why it is 42. What was your reasoning?"* 

### 3. Representation 
The framework emphasizes translating ideas across forms (words $\rightarrow$ symbols $\rightarrow$ graphs).
*Impact on Bot:* If a student struggles with algebraic symbols, the AI must automatically pivot to a different representation format, such as asking the student to visualize a table or physical objects.

### 4. Cross-Curricular Elements (EMK) & HOTS
*Impact on Bot:* The AI's dynamically generated word problems should explicitly rotate through EMK themes (e.g., *Sains dan Teknologi*, *Pendidikan Kewangan*, *Kelestarian Alam Sekitar*). For TP5/TP6 questions, the AI must engage Higher Order Thinking Skills (Evaluating, Creating) rather than just Applying formulas.

### 5. Technology & STEM Integration
The DSKP heavily emphasizes the use of technological tools like dynamic geometry software and spreadsheets.
*Impact on Bot:* The AI should actively suggest digital tools for visualization. For example, "Imagine plotting this in GeoGebra..." or "If we put this sequence in an Excel spreadsheet..." to bridge the gap between abstract math and technical modeling.

### 6. Interactive "Projek Mini" (Holistic TP6 Assessment)
School assessment should be holistic, taking forms like presentations and projects.
*Impact on Bot:* To achieve TP6 (Creative Non-Routine), the AI must design a chat-friendly "Projek Mini". The AI should instruct the student to use household items or observe their surroundings, construct a model, and then send a photo back to the AI via chat. *Note: This requires robust image processing capabilities from the underlying LLM.*

## Next Steps
This document is for the Education Lead and engineering team to review. If approved, these directives should be merged into `pai-bot`'s core system architecture prompts (e.g. `docs/implementation-guide.md`).
