# Donation Page

This is a custom donation page for myself, but you can also use it if you want.

## Features

- ✅ Easy API to connect custom tools with this
- ✅ A permission system for applications
- ✅ PayPal checkout (including credit cards)
- ✅ Connecting Donor's discord account
- ✅ Multiple funds
- ✅ Fund Goals
- ✅ Fund aliases for quick links
- ✅ Etc

## Config

This project is configured in 2 files - `.env` and `auths.json`

The `.env` file is for settings that will not be changed at runtime, and are options for setting the app up in general. To configure this, you can take a look at `template.env`, which has explanations for each configurable value. If there is a value already set (such as `PORT="3000"`), then the value that is set is the default value. This applies to empty values as well.

The `auths.json` file is for setting up external tools, allowing API connections. This file should contain a list of all applications. Note, each application **must** have a different token & name!

```json
[
  {
    "name": "Discord Bot",
    "permissions": 8,
    "token": "abc123"
  },
  {
    "name": "Twitch Relay",
    "permissions": 18,
    "token": "cba321"
  }
]
```

The permission value allows you limit the control of a certain application. For example, you don't want a twitch relay to create a new fund, do you? The permissions are as follow:

| Permission                              | Value |
| --------------------------------------- | :---: |
| Fetch the Discord Token                 |   2   |
| View life notifications                 |  16   |
| Admin fund control (create, edit, etc)  |  32   |
| Fetch historical donations              |  64   |
| Admin Permissions (ie. all permissions) |   8   |

To get the final permission value, simply add the values of all the permissions you want to add. For example, for the `Twitch Relay` app, I have it the permissions to view live notifications, and to fetch the discord token used by the donation API.

Finally, note the last permission, *8*. If an application has this, it will be able to execute any of the permissions above it. Consider it more of a shortcut.

## Usage



## Tools

