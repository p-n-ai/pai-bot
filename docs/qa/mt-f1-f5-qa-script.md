---
title: "MT F1-F5 Telegram QA Script"
summary: "Detailed QA testing scenarios for KSSM Mathematics Form 1-5, focusing on pedagogical quality, correctness, and multilingual support."
read_when:
  - You are performing QA on the Telegram bot.
  - You need to verify curriculum alignment for MT F1-F5.
  - You are testing the bot's ability to handle BM, EN, and Mixed language.
---

# MT F1-F5 Telegram QA Script

This document provides a structured set of scenarios to test the AI Tutor's performance in Mathematics (Form 1 to Form 5). 

---

## 📊 Test Case Summary
| Pattern Name | Description | Count |
|---|---|---|
| **Concept Check** | Testing definitions, properties, or basic "what is" understanding. | 14 |
| **Skill Application** | Standard procedure or calculation testing. | 16 |
| **Misconception Test** | Student provides an incorrect answer to test bot's correction logic. | 8 |
| **Formation Request** | Asking the bot to translate text to math or form a model. | 5 |
| **Verification** | Student asks the bot to check their work or final answer. | 6 |
| **KBAT / HOTS** | Real-world application or higher-order analytical questions. | 8 |
| **Navigation / Syllabus** | Inquiries about chapters, subtopics, or learning paths. | 8 |
| **Analogy Request** | Asking for a real-life comparison to simplify a concept. | 4 |
| **Step-by-Step Request** | Asking for a guided walkthrough of a problem. | 4 |
| **Example Request** | Asking for a specific problem example to practice. | 4 |
| **Total** | | **77** |

---

## 🎯 Testing Objectives
1. **Pedagogical Alignment**: Ensure the bot follows the **Dual-Loop** pattern (Understand → Plan → Solve → Verify → Connect).
2. **Correctness**: Verify mathematical accuracy and adherence to KSSM DSKP standards.
3. **Response Quality**: Confirm the bot uses the "Stop and Prompt" method (asking questions instead of just giving answers).
4. **Multilingual Flexibility**: Test how well the bot handles English (EN), Bahasa Melayu (BM), and Mixed (Manglish) inputs.

---

## 🛠️ Tester Setup
1. Open the Telegram bot.
2. Type `/start` to initialize the session.
3. Use `/language` to switch between English and Bahasa Melayu where appropriate, or simply start typing in your preferred language.
4. (Optional) Set a goal using `/goal Saya nak kuasai Algebra`.

---

## 📋 Quality Checklist
During testing, check if the bot:
- [ ] **Greets** the student warmly and establishes the context.
- [ ] **Identifies** the core problem before jumping to calculations (Understand loop).
- [ ] **Proposes a plan** and asks the student for the first step (Plan loop).
- [ ] **Provides analogies** or hints when the student is stuck.
- [ ] **Detects misconceptions** (e.g., forgetting to flip symbols in inequalities).
- [ ] **Connects** the topic to real-life or other mathematical concepts (Connect loop).

---

## 🧪 Detailed Test Scenarios

### 🧭 Navigation & Curriculum Inquiries
Test if the bot can help students navigate the syllabus and understand their learning path.

| ID | Topic | Student Input (Language) | Pattern / TP | Expected Bot Behavior | Actual Answer | Result |
|---|---|---|---|---|---|---|
| **NAV-01** | Syllabus Overview | "Tingkatan 1 belajar apa je untuk Math?" (BM) | Navigation | Should list the main chapters for Form 1. | Bagus, anda Tingkatan 1. Sekarang hantar topik atau soalan matematik yang anda mahu belajar. Saya kena ikut silibus yang ada dalam konteks sekarang: topik yang sedang jelas untuk anda ialah Sains Tingkatan 1, bukan senarai penuh Matematik Tingkatan 1.

Kalau anda nak, saya boleh bantu bagi:
senarai topik Sains Tingkatan 1, atau
anda bagi subjek Matematik Tingkatan 1 yang anda nak, saya semak ikut bab satu-satu. | **Pass (Delayed)**: Initially confused with Science. Listed Math chapters only after being re-centered on Math topics. |
| **NAV-02** | Subtopics | "What are the subtopics for Form 2 Chapter 2?" (EN) | Navigation | Should list subtopics for MT2-02. | Form 2 Chapter 2 in KSSM Math is usually about factorisation / integers-related topic? But I don’t want to guess and mislead you.

Can you tell me the subject first:
Math, Science, or another subject?

If you mean Mathematics, I can list the subtopics for the exact Chapter 2 once you confirm the title. | **Fail**: Asked for subject confirmation (Math/Science) instead of listing subtopics directly. |
| **NAV-03** | Learning Objectives| "I nak tau apa yang I patut pass untuk Bab 1 F3?" (Mixed) | Navigation | Should list the Learning Objectives for MT3-01. | Kalau Bab 1 F3, yang patut “pass” dulu biasanya benda asas bab tu sendiri, bukan terus soalan susah.

Untuk start, kamu kena boleh:
faham idea utama bab
kenal istilah penting
buat langkah asas dengan betul
jawab soalan mudah tanpa tersangkut

Kalau untuk Sains Form 3, Bab 1 biasanya saya boleh bagi ikut topik yang tepat. Bab 1 yang kamu maksudkan subjek apa? | **Fail**: Provided generic AI filler instead of specific KSSM objectives (Indices). Repeatedly asked for subject confirmation. |
| **NAV-04** | Prerequisites | "Before I start Linear Inequalities, what do I need to know first?" (EN) | Navigation | Should identify Linear Equations (MT1-06) as prerequisite. | Before Linear Inequalities, you should be comfortable with a few basics first:

Can you solve simple linear equations?
Example: 2x + 3 = 11

Do you understand number lines and comparison symbols?
<, >, <=, >=

Can you rearrange simple expressions?
Like moving terms across carefully

Do you know how to handle negatives?
Especially when multiplying or dividing by a negative number | **Pass**: Correctly identified Linear Equations and handling negatives as key prerequisites. |
| **NAV-05** | Motivation/Why | "Kenapa kena belajar Matriks ni? Susah la." (BM) | KBAT / HOTS | Provide real-life application (logistics/graphics). | Sebab matriks ni sebenarnya alat ringkas untuk susun dan kira maklumat dengan cepat.

Bayang macam jadual stok kedai: buku, pensel, pemadam. Kalau tulis satu-satu memang semak. Matriks bantu kita susun data kemas dan buat operasi lebih mudah, terutama bila masuk soalan darab matriks atau cari hubungan antara nombor.

Tak perlu terus hafal semua. Untuk mula, cukup faham 3 benda:
bentuk matriks macam mana
unsur dalam matriks itu apa
bila dua matriks sama | **Pass**: Provided a relevant "jadual stok kedai" (inventory) analogy and focused on data organization. |
| **NAV-06** | Learning Path | "Habis Bab 2 F1, I patut buat Bab apa?" (Mixed) | Navigation | Suggest Bab 3 (Kuasa Dua) or Bab 5 (Algebra) depending on context. | Kalau Form 1 Matematik dan Bab 2 dah habis, biasanya kamu boleh sambung ikut turutan bab seterusnya. Tapi saya tak nak agak bab yang salah.

Kamu maksud Bab 2 subjek Math Form 1, kan? Kalau ya, saya boleh bantu susun “next chapter” ikut topik sebenar. | **Fail**: Repeated the "confirm subject" pattern and didn't provide a suggestion. |
| **NAV-07** | Summary Request | "Can you summarize what I learned in Chapter 6 F1?" (EN) | Navigation | Provide a concise summary of Linear Equations. | I can help, but I need to know which subject and which Chapter 6 you mean for Form 1, because Chapter 6 is different in Matematik, Sejarah, Sains, and other subjects.

If you mean Form 1 Sejarah, Chapter 6 is not the topic we’re on now. If you mean another subject, tell me the subject title first, and I’ll summarize it simply for you. Untuk Matematik Tingkatan 1, Chapter 6 biasanya bergantung pada buku/susunan sekolah, jadi saya tak nak tersalah topik.

Kalau kamu maksudkan Bab 6 Matematik F1 dalam KSSM, hantar tajuk bab itu dulu, contohnya “Persamaan Linear” atau “Masa dan Waktu”, nanti saya boleh summarize apa yang kamu dah belajar dengan ringkas dan senang faham. Boleh bagi tajuk babnya?  Beri rating cepat 1-5 ya. | **Fail**: Refused to summarize even after confirmation of subject; claimed KSSM ordering is school-dependent. |
| **NAV-08** | Goal Alignment | "Is this Chapter 5 important for my SPM?" (EN) | Navigation | Explain that Algebra is the foundation for almost all SPM topics. | If you’re talking about Form 1 Math Chapter 5, yes — it can be useful for later SPM topics, but it’s not an SPM chapter by itself.

The main idea is that Chapter 5 builds your base skills. Those base skills often appear again in harder SPM Math later, like:
reading graphs or data
working with patterns and relationships
handling basic calculations accurately

So the chapter matters because it trains your thinking, not because the exact same question will come out in SPM. | **Partial Pass**: Confirmed its importance as a foundation but was slightly vague on the specific topic (Algebra). |

### Form 1: Foundation
| ID | Topic | Student Input (Language) | Pattern / TP | Expected Bot Behavior | Actual Answer | Result |
|---|---|---|---|---|---|---|
| **F1-01.1** | MT1-01 (1.1) | "Apa beza integer dengan nombor bulat?" (BM) | Concept Check / TP1 | Explain negative numbers inclusion. | hereeee | **Pass**: Clear explanation highlighting negative numbers and ended with a "Stop and Prompt" question. |
| **F1-01.2** | MT1-01 (1.3) | "Solve -5 + (-3) x 2. I think the answer is -16." (EN) | Misconception / TP3 | Detect BODMAS error. | hereeee | **Pass**: Detected the BODMAS error, showed correct steps, and prompted user to reflect on the mistake. |
| **F1-01.3** | MT1-01 (1.1) | "Give me an example of an integer in real life." (EN) | Example Request / TP1| Temperature below zero or lift floors. | hereeee | **Pass**: Provided temperature and debt examples, and prompted the student for a third one. |
| **F1-02.1** | MT1-02 (2.1) | "Senaraikan faktor bagi 12." (BM) | Skill Application / TP2 | Help student find all pairs (1x12, 2x6, 3x4). | hereeee | **Partial Pass**: Listed the factors directly instead of guiding the student to find them in pairs. |
| **F1-02.2** | MT1-02 (2.2) | "Show me step-by-step how to find FSTB for 12 and 18." (EN) | Step-by-Step / TP3 | Use repetitive division or prime factors. | hereeee | **Partial Pass**: Used the "listing factors" method instead of the standard KSSM repetitive division or prime factors. |
| **F1-02.3** | MT1-02 (2.2) | "I got GSTK for 4 and 6 as 24. Is that the smallest?" (Mixed) | Verification / TP3 | Explain that 12 is smaller than 24. | hereeee | **Fail**: Significant math error. Hallucinated that 24 is the smallest common multiple despite listing 12 in both sequences. |
| **F1-04.1** | MT1-04 (4.1) | "If ratio Ali:Abu is 2:3 and Ali has 10 marbles, how many does Abu have?" (EN) | Skill Application / TP3 | Guide to find 1 unit value first. | hereeee | **Pass**: Correctly guided the student to find the 1-unit value first and left the final step for them. |
| **F1-04.2** | MT1-04 (4.4) | "Harga 3 tin susu RM12. Ayah nak beli 10 tin tapi ada RM35 je. Cukup tak?" (BM) | KBAT / HOTS / TP4 | Calculate RM40 vs RM35. | hereeee | **Pass**: Correctly calculated the unitary value and total cost, then prompted the student for the final comparison. |
| **F1-04.3** | MT1-04 (4.1) | "Explain the concept of ratio using a cooking analogy." (EN) | Analogy Request / TP1| Mixing water and rice or cordial. | hereeee | **Pass**: Used a syrup/water analogy and explained equivalent ratios (maintaining taste). |
| **F1-05.1** | MT1-05 (5.1) | "Apa maksud 'pemboleh ubah' dalam algebra?" (BM) | Concept Check / TP1 | Explain symbol representing unknown. | hereeee | **Pass**: Used the "empty container" analogy and correctly defined the concept of variables. |
| **F1-05.2** | MT1-05 (5.2) | "Simplify 3x + 5y - x + 2y." (EN) | Skill Application / TP2 | Group like terms ($2x + 7y$). | hereeee | **Pass**: Correctly simplified the expression to $2x + 7y$ (noted: switched to BM despite EN prompt). |
| **F1-05.3** | MT1-05 (5.1) | "Give me an algebraic expression for '5 less than x'." (EN) | Formation / TP2 | Expected: $x - 5$. Watch for $5 - x$ error. | hereeee | **Pass**: Correctly identified $x - 5$ and explained the logic behind the order. |
| **F1-06.1** | MT1-06 (6.1) | "Create an equation: A number plus 7 is 15." (EN) | Formation / TP2 | Expected: $x + 7 = 15$. | hereeee | **Pass**: Correctly formed the equation $x + 7 = 15$ (minor language mixing at the end). |
| **F1-06.2** | MT1-06 (6.2) | "Solve 2(x + 3) = 10. I got x = 7." (Mixed) | Misconception / TP3 | Detect error ($2x+6=10 \to 2x=4 \to x=2$). | hereeee | **Pass**: Correctly identified the error, provided clear steps to solve ($x = 2$), and prompted for reflection. |
| **F1-06.3** | MT1-06 (6.3) | "Explain Linear Equations like I'm 5 years old." (EN) | Analogy Request / TP1| Balanced weighing scale analogy. | hereeee | **Pass**: Used the "balance scale" analogy and a "stickers in a bag" example for a simple explanation. |
| **F1-07.1** | MT1-07 (7.1) | "Apa maksud x > 5?" (BM) | Concept Check / TP1 | Explain numbers larger than 5. | hereeee | **Pass**: Correctly explained that x is any number greater than 5, and clarified that 5 is excluded. |
| **F1-07.2** | MT1-07 (7.2) | "Solve -2x < 10. Is the answer x < -5?" (EN) | Misconception / TP3 | **CRITICAL**: Sign flip check. | hereeee | **Pass**: Correctly identified the sign flip requirement and provided the correct solution $x > -5$. |

### Form 2: Application
| ID | Topic | Student Input (Language) | Pattern / TP | Expected Bot Behavior | Actual Answer | Result |
|---|---|---|---|---|---|---|
| **F2-01.1** | MT2-01 (1.1) | "Describe the pattern: 2, 5, 8, 11..." (EN) | Skill Application / TP2 | Identify common difference +3. | hereeee | **Pass**: Correctly identified the +3 common difference and prompted the student for the next step. |
| **F2-01.2** | MT2-01 (1.3) | "Give me an example of a sequence in nature." (EN) | Example Request / TP1| Fibonacci in flowers or shells. | hereeee | **Pass**: Provided sunflower and leaf pattern examples and prompted the student for a third. |
| **F2-01.3** | MT2-01 (1.3) | "How to find the 100th term without listing all?" (EN) | Step-by-Step / TP3 | Introduce $n$-th term formula. | hereeee | **Pass**: Correctly introduced the $n$-th term formula and prompted the student to substitute $n=100$. |
| **F2-02.1** | MT2-02 (2.1) | "Expand (x + 3)(x - 2)." (EN) | Skill Application / TP2 | Use FOIL/Expansion steps. | hereeee | **Pass**: Used a clear step-by-step expansion method and prompted the student to continue (switched to BM despite EN prompt). |
| **F2-02.2** | MT2-02 (2.2) | "Faktorkan x^2 - 9." (BM) | Concept Check / TP3 | Difference of two squares. | hereeee | **Pass**: Correctly identified "difference of two squares" and prompted the student to find the factors. |
| **F2-02.3** | MT2-02 (2.1) | "What's an analogy for expanding brackets?" (EN) | Analogy Request / TP1| Distributing gifts or handshakes. | hereeee | **Pass**: Used a "snack pack sharing" analogy to explain the distributive property. |
| **F2-03.1** | MT2-03 (3.1) | "Make y the subject of 2x + y = 10." (EN) | Skill Application / TP3 | Isolate y. | hereeee | **Pass**: Correctly isolated $y$ to get $y = 10 - 2x$ (switched to BM despite EN prompt). |
| **F2-03.2** | MT2-03 (3.1) | "A rectangle has A = lw. If A=50, l=10, find w." (Mixed) | Skill Application / TP3 | Substitution and solving. | hereeee | **Pass**: Correctly substituted values and prompted the student to solve the final step. |
| **F2-03.3** | MT2-03 (3.1) | "Show me another example of changing the subject." (EN) | Example Request / TP2| $V=IR$ or $F=ma$. | hereeee | **Partial Pass**: Provided a simple linear equation ($3y + 4 = 19$) instead of a formulaic example with multiple variables. |
| **F2-10.1** | MT2-10 (10.1) | "Apa maksud kecerunan sifar?" (BM) | Concept Check / TP1 | Explain horizontal line. | hereeee | **Pass**: Correctly explained that a zero gradient is a horizontal line and used a "flat road" analogy. |
| **F2-10.2** | MT2-10 (10.1) | "Check my gradient: A(1, 2), B(3, 10), m = (10-2)/(3-1) = 4." (EN) | Verification / TP3 | Confirm $8/2 = 4$. | hereeee | **Pass**: Correctly confirmed the calculation and prompted the student with a follow-up challenge. |
| **F2-10.3** | MT2-10 (10.1) | "Explain gradient using a mountain hiking story." (EN) | Analogy Request / TP1| Steepness of the trail. | hereeee | **Pass**: Used a mountain hiking analogy and introduced the "rise over run" concept. |

### Form 3: Mastery
| ID | Topic | Student Input (Language) | Pattern / TP | Expected Bot Behavior | Actual Answer | Result |
|---|---|---|---|---|---|---|
| **F3-01.1** | MT3-01 (1.2) | "Simplify (2^3)^4." (EN) | Skill Application / TP2 | Power of power law. | hereeee | **Pass**: Correctly applied the power of power law to get $2^{12}$ (switched to BM despite EN prompt). |
| **F3-01.2** | MT3-01 (1.2) | "Is 2^3 + 2^4 = 2^7? I just added the powers." (Mixed) | Misconception / TP3 | Correct the addition vs multiplication law. | hereeee | **Pass**: Correctly identified the misconception and explained that index addition rules only apply to multiplication, not addition. |
| **F3-01.3** | MT3-01 (1.2) | "Give me a hard problem on indices to try." (EN) | Example Request / TP4| $3^x \cdot 9^{x-1} = 27$. | hereeee | **Partial Pass**: Provided a relatively simple indices problem ($(3^2 \times 3^4) \div 3^3$) instead of a high-level mastery problem. |
| **F3-02.1** | MT3-02 (2.1) | "Round 0.0456 to 2 sig figures." (EN) | Skill Application / TP2 | Check leading zeros (not sig). | hereeee | **Pass**: Correctly rounded to 0.046 and identified that leading zeros are not significant. |
| **F3-02.2** | MT3-02 (2.2) | "Explain Standard Form to my non-math friend." (EN) | Concept Check / TP1 | Scientific notation for big/small numbers. | hereeee | **Pass**: Correctly explained Standard Form with examples and used a "file compression" analogy. |
| **F3-02.3** | MT3-02 (2.2) | "Calculate 1.2e5 times 3.0e2. How to do this?" (EN) | Step-by-Step / TP3 | Add powers of 10. | hereeee | **Pass**: Correctly explained the step-by-step process and prompted the student for the coefficient multiplication. |
| **F3-05.1** | MT3-05 (5.1) | "Dalam segitiga bersudut tegak, sin tu apa?" (BM) | Concept Check / TP1 | SOH / Tentang-Senget. | hereeee | **Pass**: Correctly defined sine as the ratio of opposite side over hypotenuse with a simple example. |
| **F3-05.2** | MT3-05 (5.1) | "I got sin x = 1.2. Is this possible?" (EN) | Misconception / TP3 | Explain $0 \le \sin \le 1$. | hereeee | **Pass**: Correctly identified that sine values must be within [-1, 1] and prompted for student working. |
| **F3-05.3** | MT3-05 (5.1) | "Give me a real life example where sin is used." (EN) | Example Request / TP1| Height of a kite or ramp angle. | hereeee | **Pass**: Provided a practical example of measuring the height of a tree or building using angles. |
| **F3-09.1** | MT3-09 (9.1) | "Point (1, 5) on line y = 2x + 3?" (EN) | Verification / TP3 | Substitute and confirm. | hereeee | **Fail**: Contradictory logic. Started by saying "Not on the line" but then correctly showed it IS on the line and concluded it IS on the line. |
| **F3-09.2** | MT3-09 (9.1) | "How to find x-intercept for 3x + 2y = 6?" (EN) | Step-by-Step / TP3 | Set y=0. | hereeee | **Pass**: Correctly explained the method (setting y=0) and prompted the student for the next step. |
| **F3-09.3** | MT3-09 (9.1) | "A road has gradient 0.1. What does this mean?" (EN) | KBAT / HOTS / TP4 | Slope of the road (1m rise for 10m run). | hereeee | **Pass**: Correctly interpreted the gradient and explained it as a 1m rise for every 10m forward. |

### Form 4: Advanced
| ID | Topic | Student Input (Language) | Pattern / TP | Expected Bot Behavior | Actual Answer | Result |
|---|---|---|---|---|---|---|
| **F4-01.1** | MT4-01 (1.1) | "Bentuk am fungsi kuadratik tu macam mana?" (BM) | Concept Check / TP1 | $ax^2 + bx + c$. | hereeee | **Pass**: Correctly provided the standard form $f(x) = ax^2 + bx + c$ and noted that $a$ cannot be zero. |
| **F4-01.2** | MT4-01 (1.1) | "Path h = -5t^2 + 20t. When does it hit the ground?" (Mixed) | KBAT / HOTS / TP5 | Solve roots (t=0, t=4). | hereeee | **Pass**: Correctly identified h=0 and prompted the student to begin factoring to find the roots. |
| **F4-01.3** | MT4-01 (1.1) | "Does a quadratic always have two roots?" (EN) | Concept Check / TP4 | Explain 0, 1, or 2 roots. | hereeee | **Pass**: Correctly explained that quadratics can have 0, 1, or 2 roots (switched to BM despite EN prompt). |
| **F4-02.1** | MT4-02 (2.1) | "Count 1, 2, 3, 4, 10... what base is this?" (EN) | Skill Application / TP2 | Base 5. | hereeee | **Fail**: Failed to recognize the Base 5 counting pattern. Incorrectly claimed it was base 10 and asked for clarification. |
| **F4-02.2** | MT4-02 (2.1) | "Check my conversion: 13 base 10 = 1101 base 2?" (EN) | Verification / TP3 | Correct to 1101 (8+4+0+1 = 13). | hereeee | **Pass**: Correctly confirmed the conversion and verified it using place values. |
| **F4-02.3** | MT4-02 (2.1) | "Give me a base 2 addition problem to practice." (EN) | Example Request / TP3| $101 + 011$. | hereeee | **Pass**: Provided a valid binary addition problem ($1011 + 110$) and prompted the student to try it. |
| **F4-03.1** | MT4-03 (3.1) | "Apa maksud 'Jika p, maka q'?" (BM) | Concept Check / TP1 | Conditional statement. | hereeee | **Pass**: Correctly explained the conditional statement and provided a clear "Rain/Wet Road" example. |
| **F4-03.2** | MT4-03 (3.2) | "Converse of 'If x=2, then x^2=4' is 'If x^2=4, then x=2'. Right?" (EN) | Verification / TP2 | Confirm converse, but note it might be false (x could be -2). | hereeee | **Pass**: Correctly confirmed the converse and noted its falsity due to $x = -2$ (switched to BM despite EN prompt). |
| **F4-03.3** | MT4-03 (3.1) | "Show me how to form a negation using 'not'." (EN) | Step-by-Step / TP2 | Use 'not' or 'not all'. | hereeee | **Pass**: Provided clear, simple examples of negating statements using "not". |

### Form 5: Excellence
| ID | Topic | Student Input (Language) | Pattern / TP | Expected Bot Behavior | Actual Answer | Result |
|---|---|---|---|---|---|---|
| **F5-01.1** | MT5-01 (1.1) | "y varies directly as x. Write equation." (EN) | Formation / TP1 | $y = kx$. | hereeee | **Pass**: Correctly provided the equation $y = kx$ and prompted the student regarding the constant $k$. |
| **F5-01.2** | MT5-01 (1.2) | "y inversely proportional to x^2, y=2, x=3, find k." (Mixed) | Skill Application / TP3 | $k = yx^2 = 18$. | hereeee | **Pass**: Correctly set up the inverse variation equation ($y = k/x^2$) and prompted the student for the next calculation steps. |
| **F5-01.3** | MT5-01 (1.1) | "Give me a KBAT problem about variations." (EN) | KBAT / HOTS / TP5 | Volume vs Pressure (Boyle's Law) or Salary vs Hours. | hereeee | **Pass**: Provided a relevant KBAT problem about inverse variation (speed and time) and prompted for the first part. |
| **F5-02.1** | MT5-02 (2.1) | "Can I add 2x2 with 2x3 matrix?" (EN) | Concept Check / TP1 | No, orders must match. | hereeee | **Pass**: Correctly identified that matrix addition requires the same order and explained why the given matrices cannot be added. |
| **F5-02.2** | MT5-02 (2.2) | "How to find determinant for [[a,b],[c,d]]?" (EN) | Concept Check / TP2 | $ad - bc$. | hereeee | **Pass**: Correctly provided the determinant formula $ad - bc$ with a clear mnemonic. |
| **F5-02.3** | MT5-02 (2.2) | "Solve [[2,1],[4,3]] X = [[1,0],[0,1]]." (Mixed) | Skill Application / TP4 | Matrix inversion. | hereeee | **Partial Pass**: Correctly calculated the inverse matrix but failed the "Stop and Prompt" pedagogical rule by providing the full solution immediately. |
| **F5-04.1** | MT5-04 (4.1) | "Cukai pintu vs cukai tanah?" (BM) | Concept Check / TP1 | Local council vs State land. | hereeee | **Pass**: Correctly explained the difference between Assessment Tax and Quit Rent and who collects them. |
| **F5-04.2** | MT5-04 (4.1) | "Hitung cukai jika pendapatan bercukai RM45,000." (BM) | KBAT / HOTS / TP4 | Tax bracket calculation. | hereeee | **Pass**: Correctly identified the bracket-based nature of tax calculations and requested the relevant tax table to ensure accuracy. |
| **F5-04.3** | MT5-04 (4.1) | "What happens if I don't pay tax?" (EN) | KBAT / HOTS / TP4 | Fines, legal action. | hereeee | **Pass**: Correctly identified the consequences of tax non-payment (penalties, interest, legal action). |

---

## 🎮 Feature Testing Scenarios

| ID | Feature | Input | Expected Behavior | Actual Answer | Result |
|---|---|---|---|---|---|
| **FE-01** | Goals | "/goal Saya nak habiskan Bab 1 harini" | Bot parses the goal and confirms tracking. | hereeee | **Fail**: Failed to parse the BM goal ("habiskan Bab 1") and suggested English-only examples. |
| **FE-02** | Progress | "/progress" | Bot displays a progress summary with Unicode bars/stars. | hereeee | **Partial Pass**: Formatting is correct, but data accuracy is suspicious (showed 100% for Science topics never touched in this session). |
| **FE-03** | Challenge | "/challenge" | Bot triggers the challenge matchmaking or invites to a 5-question quiz. | hereeee | **Pass**: Correctly initiated the matchmaking process and provided status updates and tips. |
| **FE-04** | Language | Type in BM then switch to EN. | Bot should smoothly transition the language of explanation. | hereeee | **Partial Pass**: Capable of bilingual interaction but shows a heavy bias towards Bahasa Melayu. Frequently defaulted to BM for math explanations even when prompted in English. |

---

## 📝 General Notes for Tester
- **Don't provide full answers**: Try to give partial or even wrong answers to see how the bot helps you recover.
- **Mix the language**: Test how it responds to "Manglish" (e.g., "Bot, help me solve this math questions, I don't know how to do lah").
- **Check for Nudges**: If you leave the chat for a while, does the bot nudge you later to continue?
 a while, does the bot nudge you later to continue?
