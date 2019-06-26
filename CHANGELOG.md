# CHANGELOG

## 2.1.1 (26th June, 2019)

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
