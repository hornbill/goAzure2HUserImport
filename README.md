### Azure Import Go - [GO](https://golang.org/) Import Script to Hornbill

Please see [Azure User Import](https://wiki.hornbill.com/index.php/Azure_User_Import) for instructions.

Filter is optional.

Sample filter (to only return those with displayName starting with "Dav" (eg: "Dave", "Davinia" etc)):

AzureConf.Filter = "startswith(displayName,'Dav')"