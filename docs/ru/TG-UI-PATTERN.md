**RU** · [EN](../TG-UI-PATTERN.md)

# Шаблоны TG UI

**A** — список + таблица + кнопки номеров  
**B** — карточка + таблица field/value + ops  
**C** — status read-only одна таблица  

Правила: нет кнопок в `<td>`; reply=edit-first; EN/RU все строки.  
Operator hub: таблица состояния (API public, Users→SP) + **одна** toggle-кнопка на действие.

## Hub exception (in-body actions)

Hub screens (Fleet, Operator, Nodes, Sites) may place **section actions** in the message body (`tg-button-row`). **Navigation** (parent / main menu) stays on the inline keyboard. List↔card screens still require table rows and action buttons from the same visible slice.
