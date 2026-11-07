; Checkers v2.0.0 - standalone setup. Build: ISCC.exe CheckersSetup.iss
; (c) 2026 Ironite
#define GameName    "Checkers"
#define Bundle      "Checkers"
#define Exe         "Checkers.exe"
#define GameVersion "2.0.0"
#define AppGuid     "{{4C7A1E92-3B6D-4F08-9E25-D81B6A0C57F3}"
#define Art         "checkers"
#define Tagline     "Six regulars at the Hollow Creek General Store, from beginner to county champion, a friend on the same computer, or a friend online."
#define SaveWhat    "unfinished game, record against each opponent and settings"
#include "GameSetup.iss"

; Let Checkers host online games without a firewall prompt.
[Run]
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall add rule name=""Checkers"" dir=in action=allow program=""{app}\Games\{#Bundle}\{#Exe}"" enable=yes"; Flags: runhidden; StatusMsg: "Letting Checkers host online games..."

[UninstallRun]
Filename: "{sys}\netsh.exe"; Parameters: "advfirewall firewall delete rule name=""Checkers"""; Flags: runhidden; RunOnceId: "CheckersFirewall"
