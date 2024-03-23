# Apptrack (Application Tracker)

`apptrack` is a CLI tool for tracking job applications.
It automatically parses jobs from LinkedIn, Greenshouse, and Lever and adds it to a Notion database.
Can also manually fill in information for other websites.

## Installation 

Use go to install from source.
```
go install github.com/leoldding/apptrack
```

## Setup

### Database

On a new Notion page, create a database with the following property names and types:
* Company (title)
* Position (text)
* Location (text)
* Date Applied (date)
* Status (status)
    * To-do
        * Ready to apply (default)
    * In progress
        * Applied
* Link (text)

Feel free to add other properties for your own database such as `Accepted` to the `Complete` sub-property under `Status`. 

### Database ID

Make sure that you have the database opened on as a full page. Use the `Share` menu on top to `Copy link`. 
Paste the link somewhere so that you can inspect it. 

The link should be in the following format:
```
https://www.notion.so/{workspace_name}/{database_id}?v={view_id}
```

Keep track of the part in the link that corresponds to `{database_id}`.

### Integration

Follow the first three sections under `Getting Started` in this [Notion guide](https://developers.notion.com/docs/create-a-notion-integration#getting-started) to create a Notion integration, get your API key, and giving the integration permissions to your database.

### Variables

Now we will use the database ID and API key from the two previous steps.

Add `APPTRACK_NOTION_DATABASE_ID=<your-database-id>` and `APPTRACK_NOTION_API_KEY=<your-api-key>` to your `~/.bashrc` or `~./zshrc` file in order for the tool to work.

## Usage

Call the tool by running `apptrack "<your-job-link>"`. 

The tool will automatically check if the job is from **LinkedIn**, **Lever**, or **Greenhouse**.
If it is from one of those three boards, the tool will attempt to automatically scrape the information.
If there is any information that was not found, you will be prompted to fill it in.

If the link is from another website, you will simply be prompted to fill in the necessary information.

### Manual Input 

If you want to manually input information for any job link, use the `-manual` or `-m` flag and you will be promptedy to fill in information.

### Saving Jobs

If you haven't applied to a job, you can use the `-save` or `-s` flag. 
It will set the `Status` property to the default value of `Ready to apply` and not `Applied`.
