# 🍺 Brewbot Commands

## 📅 Scheduling
**/propose** `dates`
Propose one or more dates for the next brew day. Comma-separate multiple dates.
Example: `/propose March 15, March 22, April 5`

**/startpoll**
Start a vote from all proposed dates. Everyone reacts with their choice — first to hit majority wins.

**/closepoll**
Manually close the poll and pick the winner (highest votes).

---

## 🔄 Rotation
**/rotation list**
Show the full brewer rotation with 👉 pointing at who's up next.

**/rotation add** `@user`
Add someone to the end of the rotation.

**/rotation skip** `@user` `[reason]`
Skip a brewer this round — moves them to the end of the queue.

**/rotation next**
Show who is brewing next without any changes.

---

## 🧪 Recipes & Ratings
**/recipe submit** `name` `[style]` `[og]` `[fg]` `[ingredients]` `[notes]`
Submit the recipe for your brew. **Renames the channel** to your brew name.
Run this in your brew channel.

**/recipe view**
View the submitted recipe for the current brew channel.

**/rate** `1–5` `[notes]`
Rate the current brew with tasting notes. Run this in the brew channel after the session.

**/complete**
Mark the brew as complete and add it to the blackboard.

---

## 📋 Blackboard & Utilities
**/blackboard**
Show all past brews with brewers, styles, ABV, and ratings.

**/abv** `og` `fg`
Calculate ABV and apparent attenuation from original and final gravity.
Example: `/abv og:1.065 fg:1.012` → **6.9% ABV**

---

## 📖 Typical Flow
1. Everyone runs `/propose` with their available dates
2. Someone runs `/startpoll` — bot posts a vote
3. React to vote — majority closes it automatically
4. Bot creates a brew channel and pings the brewer
5. Brewer runs `/recipe submit` in their channel
6. After the session, everyone runs `/rate`
7. Brewer runs `/complete` to archive it to the blackboard
