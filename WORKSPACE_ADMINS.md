# `gro` for Google Workspace admins

A guide for setting up `gro` once for everyone in your Google Workspace org, so individual users don't need to click through Google Cloud Console themselves.

## TL;DR

If you run a Google Workspace org and want your users to use `gro` without each doing their own OAuth setup, you can create an **Internal** OAuth client in Google Cloud and distribute the resulting `credentials.json` to your users (1Password works well). Internal apps are restricted to your Workspace domain, which means Google does **not** require app verification or a CASA security assessment — both of which would otherwise apply to two of the scopes `gro` requests. The whole setup takes ~10 minutes and has no app-verification or CASA fees.

This guide is for the admin. Once you're done, your users just paste the JSON when they run `gro init` — no Google Cloud Console knowledge needed on their end.

## Why this works

`gro` requests seven OAuth scopes, two of which (`gmail.modify` and `drive.readonly`) are on Google's **restricted scope** list. Restricted scopes normally trigger:

- A multi-week Google app-verification review.
- An annual third-party CASA security assessment (paid, ongoing).
- A 100-user lifetime cap until verification clears.
- An "unverified app" warning screen for end users.

These requirements apply when the OAuth app's audience is set to **External** (any Google account can authenticate). If you instead set the audience to **Internal** (only accounts in your Workspace domain can authenticate), Google waives all of the above. The trade-off is that no one outside your Workspace org can use this OAuth client — which is fine for a CLI you distribute only to employees.

Further reading from Google:
- [When verification is not needed](https://support.google.com/cloud/answer/13464323)
- [Manage app audience (Internal vs External)](https://support.google.com/cloud/answer/15549945)
- [Restricted scope verification](https://developers.google.com/identity/protocols/oauth2/production-readiness/restricted-scope-verification)

## Prerequisites

Before you start:

1. **You are a Google Workspace admin** on the domain you want this to work for (e.g. `signalft.com`). Admin role gives you Workspace-side controls; you may also need to allowlist the OAuth client in the Admin console later if your org restricts third-party OAuth apps (see Troubleshooting).
2. **You have Google Cloud rights to create a project under the Workspace-linked Cloud organization**. The relevant IAM role is `roles/resourcemanager.projectCreator` at the organization. Workspace admin alone is not always sufficient if Cloud IAM is gated separately — check with whoever administers your org's GCP setup if you're unsure.
3. **About cost**: there are no Google app verification or CASA fees for the Internal path. Standard GCP and Workspace billing applies if your org already has it, but this walkthrough enables only free-tier APIs (Gmail, Calendar, People, Drive) and does not provision any billable services.

## Step-by-step walkthrough

### 1. Create a Google Cloud project under your Workspace org

1. Go to <https://console.cloud.google.com/>.
2. Sign in with your Workspace admin account.
3. Top-bar project picker → **New project**.
4. Name it something descriptive (e.g. `<company>-gro-cli`).
5. **Crucially**: set **Organization** to your Workspace org (e.g. `signalft.com`), not "No organization." If the org doesn't appear in the dropdown, your account isn't an org member or doesn't have project-creation rights — resolve that before continuing.
6. Click **Create** and wait for the project to provision (a few seconds).

### 2. Enable the four APIs `gro` calls

1. Left nav → **APIs & Services → Library**.
2. Search for and enable each of these (one at a time; each takes a few seconds):
   - **Gmail API**
   - **Google Calendar API**
   - **People API** (this is the Contacts/profile API — not "Contacts API")
   - **Google Drive API**
3. Verify by clicking **APIs & Services → Enabled APIs & services** — all four should be listed.

(Some other APIs may already be enabled by default at the org level — Cloud Logging, BigQuery, etc. Those are GCP infrastructure plumbing; you don't need to disable them, and `gro` doesn't use them.)

### 3. Configure the OAuth consent screen with Audience = Internal

The newer Google Cloud Console calls this area **Google Auth Platform**; older UIs call it **OAuth consent screen**. They configure the same thing.

1. Left nav → **APIs & Services → OAuth consent screen** (or **Google Auth Platform**).
2. If a "Get started" page appears, work through it:
   - **App Information**: App name = `gro` (this is what users see on the consent screen). User support email = your Workspace address.
   - **Audience**: select **Internal**. *(If "Internal" is not available, the most common cause is that the project isn't under your Workspace org — go back to step 1 and check the Organization field. Also verify you're signed in as a Workspace user and have org-level IAM permissions to edit the consent screen.)*
   - **Contact Information**: developer contact email.
   - **Finish**: agree to the Google API Services User Data Policy.

### 4. Add the seven scopes

1. Left nav → **Data Access** (older UI: a tab inside OAuth consent screen).
2. Click **Add or Remove Scopes**.
3. In the scope filter box, paste each of these scope URLs one at a time and check the box next to the match:

```
https://www.googleapis.com/auth/gmail.modify
https://www.googleapis.com/auth/calendar.readonly
https://www.googleapis.com/auth/calendar.events
https://www.googleapis.com/auth/contacts
https://www.googleapis.com/auth/userinfo.profile
https://www.googleapis.com/auth/drive.readonly
https://www.googleapis.com/auth/drive.metadata
```

4. Click **Update**, then **Save** on the Data Access page.

You will see a banner mentioning that some of these are "sensitive" or "restricted" scopes and reference verification. **Ignore it for Internal apps** — that banner is generic and Internal-audience apps skip verification. If the page physically refuses to save, your Audience setting didn't actually save as Internal in step 3.

### 5. Create the OAuth client

1. Left nav → **Clients** (older UI: **Credentials**).
2. **Create Client** → **Application type: Desktop app**. Desktop app is what lets `gro` use a `http://localhost` loopback redirect.
3. Name: anything (e.g. `gro-desktop`). Only visible in the console.
4. Click **Create**. A dialog will show the **Client ID** and **Client Secret** plus a **Download JSON** button.
5. Click **Download JSON** and save the file. This is the `credentials.json` you'll distribute. Treat it as credential-like (you'll store it where only your users can reach it), but note that Google explicitly documents that desktop-app client secrets are **not** truly secret — the OAuth flow's PKCE protection is what defends installed apps, not the client secret itself. See the [Google OAuth installed-app documentation](https://developers.google.com/identity/protocols/oauth2/native-app).

### 6. Verify by running `gro init` yourself

Before handing the JSON out, prove it works on your own account.

1. Install `gro` if you haven't (`brew install open-cli-collective/tap/google-readonly` or see the [README](./README.md)).
2. Move the downloaded JSON into place:
   ```bash
   mv ~/Downloads/client_secret_*.json ~/.config/google-readonly/credentials.json
   ```
   (Create the directory first if needed: `mkdir -p ~/.config/google-readonly`.)
3. Run `gro init` and complete the OAuth flow in your browser.
4. Expected: a normal Workspace consent screen with your org name, no "Google hasn't verified this app" warning, all seven scope descriptions visible. Click **Allow**.
5. After the redirect (which will hit a `localhost` URL that may look like a connection error — that's expected), the terminal should print `Token saved to Keychain` and `Verified Gmail API for <you>@<your-domain>`.
6. Try `gro me` and `gro mail list --max 3` to confirm it actually works.

If step 4 shows an "unverified app" warning instead of going straight to consent, the Audience accidentally got saved as External — revisit step 3.

## What you've authorized vs. what `gro` actually does

The consent screen wording comes from Google's static scope descriptions, which describe the *maximum capability* the scope can grant. `gro` does not use the full capability of every scope.

| Scope | Consent-screen wording | What `gro` actually does |
|---|---|---|
| `gmail.modify` | "Read, compose, **and send** emails from your Gmail account" | Read, search, archive, star, label, mark read/unread, draft (compose-only — drafts are never sent automatically). **No send, no delete, no trash.** |
| `calendar.readonly` | "See and download any calendar you can access using your Google Calendar" | List calendars and read events |
| `calendar.events` | "View and edit events on all your calendars" | RSVP and color-code events; no calendar settings changes |
| `contacts` | "See, edit, download, **and permanently delete** your contacts" | Read contacts and groups; manage group membership and starring. **No delete.** |
| `userinfo.profile` | "See your personal info, including any personal info you've made publicly available" | Read the authenticated user's name and email (powers `gro me` and `gro init` verification) |
| `drive.readonly` | "See and download all your Google Drive files" | List, search, get metadata, download file content |
| `drive.metadata` | "View and manage metadata of files in your Google Drive" | Star/unstar files. No file content changes. |

Two of these scope descriptions overstate what `gro` does — `gmail.modify` includes "send" and `contacts` includes "permanently delete." These are restrictions Google offers as separate sub-scopes only for sensitive but not restricted scopes, so `gro` has to request the broader scope to get the parts it does use. The non-destructive guarantee comes from the code, not the scope:

- **Structural CI guardrails**: `internal/architecture/architecture_test.go` runs at every CI build and fails if any source file in the repo contains one of an explicit list of destructive Google API method patterns (`.Send(`, `.Untrash(`, `.BatchDelete(`, and others). This is a guardrail that catches the named patterns; it is not a proof that every conceivable destructive call is impossible.
- **No destructive command paths**: `gro`'s top-level commands (under `internal/cmd/`) do not expose `send`, `delete`, `trash`, or equivalent operations. Compliance reviewers can audit `internal/cmd/` directly to confirm what the binary surfaces to users.

Together — the scope list, the structural guardrails, and the absent destructive commands — these are what back the "non-destructive" promise. If your compliance review asks "why does the consent screen say 'send' if `gro` is read/organize-only?", the answer is: Google's scope is broader than what `gro` uses; the code's structural guardrails and the absence of any send-class command are the actual constraint.

## Distributing `credentials.json` to your users

### Recommended: 1Password shared vault

1. Create an item in an org-shared 1Password vault (e.g. "Engineering").
2. Attach the `credentials.json` as a document, or paste its contents into a Secure Note field.
3. Tell users: download the document (or copy the Secure Note contents) and put it at `~/.config/google-readonly/credentials.json`, then run `gro init`.

### MDM-pushed file

If your org uses an MDM solution (Jamf, Kandji, Intune), pushing `credentials.json` to `~/.config/google-readonly/credentials.json` during onboarding works well.

### Other channels (with caveats)

- **Slack pinned message** or **shared Drive folder**: acceptable *only* if the access list of that channel/folder matches who should be using `gro`. Don't post the file in a `#general`-style channel — even though desktop-app secrets aren't truly secret, the credentials shouldn't leak outside your org.
- **Internal HTTPS endpoint**: serving the JSON from an internal URL behind SSO is fine if you already have that infrastructure. Not worth building just for this.

## Maintenance

### Rotating the client secret

If you suspect the JSON has leaked outside your org (it ended up in a public Slack, a former employee took a copy, etc.):

1. Google Cloud Console → APIs & Services → **Credentials** → click your OAuth client.
2. **Reset secret**. This invalidates the old secret immediately.
3. **Download JSON** to get the new one and re-distribute via your channel of choice.
4. Active user tokens issued before the rotation continue to work until they're revoked or expire — see below to force revocation.

### Revoking a specific user's access

A user who has already authorized `gro` can revoke their own token from <https://myaccount.google.com/permissions> → find the app name → **Remove access**. Admins can revoke org-wide:

1. Google Workspace Admin Console → **Security → Access and data control → API controls → Manage Third-Party App Access**.
2. Find the `gro` OAuth client by name.
3. Block it (revokes all org-user tokens) or scope its access more narrowly.

### Monitoring usage

Google Cloud Console → APIs & Services → **Credentials** → click your OAuth client. The metrics tab shows authorization counts. The **APIs & Services → Dashboard** view shows per-API request volumes.

## Troubleshooting

### Users see "Access blocked: This app is blocked"

Your Workspace admin policy is restricting third-party OAuth apps. In Admin Console → **Security → API controls → Manage Third-Party App Access**, find or add the `gro` OAuth client ID and set it to **Trusted** (or at least allow the specific scopes).

### Users see "Google hasn't verified this app"

The audience accidentally got set to **External** instead of **Internal**. Go back to the OAuth consent screen and switch it back to Internal. (If a user is using a personal `@gmail.com` account rather than their Workspace account, that's the cause instead — Internal apps only accept Workspace accounts in the configured domain.)

### Tokens expire after 7 days

That's the External-app testing-mode behavior, not Internal. Same fix as above: check your audience is Internal, not External-in-Testing.

### "Internal" isn't available as a User Type

Most often: the GCP project isn't under your Workspace org (set during project creation). Also check that you're signed in as a Workspace account, not a personal Google account, and that you have org-level IAM permissions to edit the consent screen.

## FAQ

**Can I limit which Workspace users get access?**
Yes. Use the Admin Console's third-party OAuth allowlist to restrict access to specific OUs (organizational units) or groups. The OAuth client itself can't enforce subgroup restrictions, but Workspace can gate access at the admin level.

**What if I want to share this with a partner org?**
You can't, with Internal audience — Internal locks to a single Workspace domain. Either the partner org sets up their own Internal OAuth client following this same guide (recommended), or you switch your audience to External and go through Google's full verification process (CASA, fees, multi-week review). Don't do that for a partner-org use case; have them stand up their own.

**Does `gro` send my users' data anywhere?**
No. `gro` is a local CLI. It talks directly from the user's machine to Google's APIs using their OAuth token. No data is sent to any third-party server. Tokens are stored locally (macOS Keychain, Linux libsecret, or a `0600` file as fallback).

**What happens if Google changes their restricted-scope policy?**
Internal-audience apps have historically been exempt from app verification (this is a long-standing Google policy, not a per-app concession). If that changes, you'd see a notice in the Cloud Console and have a transition window. Worst case, you can fall back to having users do their own DIY setup (the path described in the main README).

**Is the client secret in `credentials.json` actually secret?**
For desktop-app OAuth clients, no — Google's documentation explicitly says the client secret is not truly secret, and the OAuth flow's PKCE mechanism is what defends installed apps. That said, treat the file as access-controlled: it identifies your org's OAuth client and shouldn't be shared outside your user population.

## Related docs

- [README.md](./README.md) — main `gro` documentation, including the DIY (non-admin) setup path
- [docs/architecture.md](./docs/architecture.md) — codebase structure
- [docs/golden-principles.md](./docs/golden-principles.md) — structural rules enforced at CI time, including the non-destructive guardrails
