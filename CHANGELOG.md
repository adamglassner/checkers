# Changelog

All notable changes to Checkers are recorded here.
Versions follow Semantic Versioning (MAJOR.MINOR.PATCH).

## [2.0.0] - 2026-11-07

### Changed
- Checkers is now a native Go game, built with Ebitengine. It no longer needs Microsoft Edge to run, starts faster and uses less memory. The boards, pieces, portraits, music and sound effects are the same as before.
- The board and every panel are drawn at your screen's full resolution, from 720p up to 4K, and the window can be resized freely (F11 for full screen).
- The six regulars get a new name each time you sit down to a game, always one that fits who they are (Little Tucker might be Little Billy next time, Miss Opal might be Miss Pearl). Your record against each of them is kept by level.

### Added
- Online play: one player hosts and a friend joins with the host's address (TCP port 47020). Each picks a name and a face. The host chooses the colours and whether jumping is compulsory, and their game referees every move. You can offer a draw, resign or ask for a rematch (the colours swap). If your friend leaves mid-game, you can finish against the computer. Your online record is kept separately.
- Same Computer games now have names and faces for both players, shown on the plaques, the score sheet and the result.
- The setup lets Checkers host online games without a Windows Firewall prompt.

### Kept
- Your unfinished game, your record against each regular and your settings carry over from 1.0.0 (still in %APPDATA%\Ironite\Checkers).

## [1.0.0] - 2026-10-28

### Added
- First release of Checkers for Windows.
- Full American checkers rules: diagonal moves, compulsory captures with multi-jumps, crowning on the far row, kings that move both ways, and a loss when you have no pieces or no move. Draws by threefold repetition, the forty-move rule or agreement.
- Play the computer or a friend. Six regulars at the Hollow Creek General Store will give you a game, from Little Tucker (just learning) through Miss Opal, Deacon Earl, Hattie Mae and Doc Whitfield to Old Gus, county champion.
- Play as Black or Red, or let a coin toss decide. Jumping can be compulsory (standard) or optional (house rules).
- Click-to-move or drag-and-drop. Multi-jumps can be taken one landing at a time or in one go. Pieces that must jump glow gold.
- Undo, Hint, Flip Board, Offer Draw and Resign.
- A score sheet in standard 1-32 square notation that lets you step back through the game and copies the moves as text. Square numbers can be shown on the board.
- Your unfinished game is waiting when you come back, and your record against each opponent is kept, in %APPDATA%\Ironite\Checkers.
- HD graphics drawn at your screen's full resolution, from 720p up to 4K: three boards (Barrelhead, Parlor Red and Tournament Green), three sets of pieces (Red & Black, Walnut & Maple and Bottle Caps) with stacked, crowned kings, on a gingham cloth in an old country store.
- Four original old-time tunes: "Potbelly Stove Waltz", "Kings Row Breakdown", "Penny Candy Rag" and "Sweet Tea on the Porch", with volume control and mute.
- Sound effects for pieces on the board, each hop of a jump, captures, crowning and the end of the game. Bottle caps clink.
