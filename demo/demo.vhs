# https://github.com/charmbracelet/vhs/

Output demo.webp

Set FontSize 26
Set Padding 12
Set Width 1400
Set Height 80
Set Theme "Builtin Light"
Set TypingSpeed 25ms
Set Shell zsh

Type "ec2ssh my-app-server  # connect by name tag, IP, or instance ID"
Sleep 1s
Hide
Type "clear"
Enter
Show

Type "ec2ssm my-app-server  # shell via SSM (no SSH needed)"
Sleep 1s
Hide
Type "clear"
Enter
Show

Type "ec2ssh --use-eice my-private-server  # tunnel via EICE"
Sleep 1s
Hide
Type "clear"
Enter
Show

Type "ec2ssh --use-ssm my-private-server  # tunnel via SSM"
Sleep 1s
Hide
Type "clear"
Enter
Show

Type "ec2scp ./data.tar.gz user@my-server:/backup  # scp and sftp supported"
Sleep 1s
Hide
Type "clear"
Enter
Show

Type "ec2list --profile prod  # list instances"
Sleep 1s
