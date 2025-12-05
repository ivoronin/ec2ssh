# https://github.com/charmbracelet/vhs/

Output demo.webp

Set FontSize 26
Set Padding 12
Set Width 1660
Set Height 80
Set Theme "Builtin Light"
Set TypingSpeed 25ms
Set Shell zsh

Type "ec2ssh -l ec2-user i-0e2d55b1a6328d2ea  # connect using generated ephemeral SSH key"
Sleep 1s
Hide
Type "clear"
Enter
Show

Type "ec2ssm i-0e2d55b1a6328d2ea  # connect through SSM (no SSH)"
Sleep 1s
Hide
Type "clear"
Enter
Show

Type "ec2ssh --use-eice i-0e2d55b1a6328d2ea  # tunnel through EC2 Instance Connect Endpoint or SSM"
Sleep 1s
Hide
Type "clear"
Enter
Show

Type "ec2scp ./file.txt ec2-user@i-0e2d55b1a6328d2ea:/data  # use scp (or sftp) to transfer files"
Sleep 1s
Hide
Type "clear"
Enter
Show

Type "ec2ssh --profile dev --region us-east-1 i-0e2d55b1a6328d2ea  # use multiple accounts and regions"
Sleep 1s
