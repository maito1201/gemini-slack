# gemini-slack
integration slack and Gemini API on GCP

## usage
deploy app as GCP Cloud Functions

```
cd ./gemini-slack

gcloud functions deploy GeminiSlack --runtime go120 --trigger-http --region asia-northeast1 --env-vars-file env.yml --entry-point GeminiSlack --source ./gcp
```

NOTE: setup env.yml as below

```env.yml
BOT_USER_TOKEN: "xoxb-your-bot-token"
GEMINI_API_KEY: "your-api-token"
```
get GEMINI_API_KEY from https://makersuite.google.com/app/apikey
