# Roadmap

This provider ships only what has a verified, documented public Developer
Hub API behind it (see `DESIGN.md` §5). The features below were researched
and are known to be missing a confirmed public API in this repository's
network-restricted research pass (`DESIGN.md` §13) - they are the priority
list for whoever picks this up next, in order:

1. **Domain categories.** Highest practitioner value per the original
   project brief; needs the actual `api.sap.com` "APIMgmt" package
   artifacts (`DESIGN.md` §5) verified, which this repository's sandbox
   could not reach.
2. **Access governance and subscription governance settings.** Both
   confirmed to exist as tenant settings; no confirmed write API yet.
3. **Bulk/admin user registration** via the named
   `DevPortal_RegisteringUsers_CF` artifact - the artifact is confirmed to
   exist, its schema is not.
4. **Site configuration/branding**, if a public API surfaces for it.
5. **Centralized Developer Hub connections**, if a public write API
   surfaces - with mandatory irreversibility warnings in the resource docs
   (`DESIGN.md` §10) and no forced recreation.
6. **Rate plans** (`APIMgmt.RatePlans`), if SAP Monetization support is
   ever in scope for this provider - the entity and its navigation
   property are already confirmed to exist (`DESIGN.md` §9/§12).

(Application `app_key`/`app_secret` were originally on this list; a later
research pass found the exact field names and they're implemented as of
`developerhub_application` - see `DESIGN.md` §6/§18.)

None of these should be implemented against a guessed endpoint - update
`DESIGN.md` §5 with the verified evidence first, the same way every
currently implemented resource is documented there.
