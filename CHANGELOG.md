# CHANGELOG

## 2.6.1

Changes:

- Moved from thumbnailPhoto to photo. Legitimate Azure Image sizes are: "48", "64", "96", "120", "240", "360", "432", "504", and "648" - please note they are all strings - IF set to "tn" it will pick up thumbnail as before (the new configuration file is defaulted to this to be on the safe side)
- businessPhones is coming back as ["#phone#"] (a so-called "string collection"), so now stripping the phone number fields from any potential square bracket and double-quotes. Only "Phone" and "WorkPhone" (Profile) have been "treated" in this way. AzureConf.StringCollectionTweak needs to be set to true use.

Fixes:

- Fixed issue with Employee ID not applying because a serverBuild was incorrectly done

## 2.6.0

Changes:

- Added support to define if users should be created, updated or both

## 2.5.1 (June 12th, 2020)

Changes:

- minor changes to be compatible with new crosscompile script

## 2.5.0 (May 28th, 2020)

Fix:

- Re-added support to define additional fields to return from Azure and map into Hornbill user fields and groups
- Fixed issue with group memberships not applying
- Fixed issue with Home Organisation only being set on user record update
- Re-added SetAsHomeOrganisation back into default conf.json
- Fixed issues with profile attributes only being set on user record update
- Fixed issues with site not always being imported
- Fixed issues with incorrect errors being reported
- Fixed issues with badly handled nil values
- Fixed issues with manager records not always being imported 
- Tidied up code & removed references to other tools whose code were used to build this one

## 2.4.2 (May 14th, 2020)

Fix:

- Fix to Group selection - only last Azure results from last selected group would be processed

## 2.4.1 (April 15th, 2020)

Change:

- Updated code to support Core application and platform changes

## 2.4.0 (January 9th, 2020)

Changes:

- Added support for new Login ID ands Employee ID fields in user record

## 2.3.0 (23rd October, 2019)

Changes:

- Reworking to match LDAP imports - but with local configuration file
- PLEASE NOTE the CONFIGURATION file has changed significantly.
- Added feature to allow the setting of a Home Organisation when creating/updating users

## 2.2.1 (4th July, 2019)

Features:

- Added additional logic to avoid 0 length password requests
- Added more debug logging around reading password profile from instance

## 2.2.0 (3rd July, 2019)

Features:

- Added debug mode to output more detailed logging

## 2.1.1 (26th June, 2019)

Fixes:

- Fixed issue where generating random password string would fail if requested length of password was less that the sum of the minimum character type settings
  
Changes:

- Updated minimum password length to 16

## 2.1.0 (29th May, 2019)

Features:

- General tidy-up of code
- Enable additional Azure properties to be returned when searching for Azure Users, and Group Members
- Updated user password generation code to enforce Hornbill instance user password profile settings
- Removed requirement to provide instance zone
- Added code to remove user images from session once attached
- Reduced number of sessions required
- Improved logging output for API calls
- Grouped user log entries when using concurrency

## 2.0.2 (22nd May, 2019)

Fixes:

- Fix to paging

## 2.0.1 (21st May, 2019)

Fixes:

- Fix to paging

## 2.0.0 (10th May, 2019)

Features:

- now using Microsoft Graph API (as opposed to Azure AD Graph API)

## 1.4.4 (8th May, 2019)

Fixes:

- Fixing Manager Lookup
  
## 1.4.3 (26th September, 2018)

Fixes:

- Recoding to use entityBrowseRecords2 instead of entityBrowseRecords.
  
## 1.4.2 (5th January, 2018)

Fixes:

- Amended an error message.

## 1.4.1 (1st December, 2017)

Fixes:

- Amended template processing at text only instead of HTML.

## 1.4.0 (21st November, 2017)

Features:

- Incorporated paging to go beyond default 100 item search limit.

## 1.3.0 (13th November, 2017)

Features:

- Added ability to import Users from specified Azure Groups.
- Refactored code to make the project easier to read and maintain.

## 1.2.0 (April 28th, 2017)

Features:

- Added ability to link in Thumbnails.

## 1.0.0 (April 5th, 2017)

Features:

- Initial Release
