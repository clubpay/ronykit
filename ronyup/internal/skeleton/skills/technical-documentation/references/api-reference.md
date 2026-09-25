# API Reference, Docstrings, and CLI Help

Reference material is descriptive, complete, and formulaic on purpose. Readers arrive at a reference entry mid-task, read one entry, and leave. This file owns rules **A1–A7**.

**Contents**
1. [What to document (A1)](#what-to-document-a1) · every public element, and what "public" means per language
2. [Method descriptions: verb by category (A2)](#method-descriptions-verb-by-category-a2)
3. [Parameters (A3)](#parameters-a3) · non-boolean, boolean, optional, units
4. [Return values and exceptions (A4)](#return-values-and-exceptions-a4)
5. [Deprecations (A5)](#deprecations-a5)
6. [Complete example: `StorageClient`](#complete-example-storageclient) · TypeScript, plus a Go mirror
7. [Reference voice (A7)](#reference-voice-a7)
8. [CLI help text (A6)](#cli-help-text-a6-convention) and [REST reference pages](#resthttp-reference-pages-convention) — both convention

---

## What to document (A1)

Document every exported type, interface, function, method, field, constant, and enum value (in TypeScript, every exported symbol). An undocumented public element reads as unsupported: developers skip it, file bugs against it, or reimplement it. A reader can't tell "not documented" from "not there".

"Public" is defined by the language, not by intent (convention — per-language norms, not the style guide):

| Language | Public surface |
|----------|----------------|
| TypeScript / JavaScript | Exported from the package entry point, including types and enum members |
| Python | Names without a leading underscore, or listed in `__all__` |
| Rust | `pub` items reachable from the crate root, including fields and variants |
| Go | Identifiers starting with a capital letter, including package-level errors |
| Java / C# | `public` and `protected` members of public types |

Write the summary as one sentence, first, in the entry's own paragraph. The summary answers "what does this do"; it never restates the name. A second paragraph, when one is needed, adds what the reader can't infer from the signature: side effects, cost, lifecycle, concurrency safety, ordering guarantees, or a link to the task page.

**Before → after (class summary):**

- Before: `/** StorageClient class. Used for storage. */`
- After: `/** Reads and writes objects in a single storage bucket. */` followed by a second paragraph: `A client opens one connection per instance and reuses it. Create one client per bucket and share it across requests; the client is safe for concurrent use.`

**Before → after (enum value):**

- Before: `ARCHIVE, // archive`
- After: `ARCHIVE — Lowest storage price, highest retrieval price. Intended for objects read less than once a year.`

Source: api-reference-comments

---

## Method descriptions: verb by category (A2)

Open a method description with a third-person present-tense verb chosen by the method's category. The verb tells the reader the shape of the call before they read the parameters. `[EN]` The exact verb wordings below are English; the category-to-verb discipline applies in any language.

| Category | Opening verb | Example first sentence |
|----------|--------------|------------------------|
| Boolean getter | Checks whether… | `Checks whether the bucket has an active retention policy.` |
| Other getter | Gets the… | `Gets the storage class of the bucket.` |
| Setter | Sets the… | `Sets the retention period, in days, for objects in the bucket.` |
| Creator or factory | Creates a… | `Creates a signed URL that grants temporary read access to an object.` |
| Everything else | Returns / Registers / Sends / Deletes / Validates / Uploads… | `Deletes the object and every one of its versions.` |

Drop the "This method…" and "This function…" openers, along with "A function that…" and "Method to…". The entry already appears under the member's name and signature, so the phrase spends the reader's first four words on the heading.

**Before → after:**

- Before: `/** This method is used for getting the customer associated with a subscription. */`
- After: `/** Gets the customer that owns the subscription. */`

- Before: `/** Function that checks if a bucket is public or not. */`
- After: `/** Checks whether anyone with the URL can read objects in the bucket. */`

- Before: `/** Will create a new signed URL for the object and return it to the caller. */`
- After: `/** Creates a signed URL that grants temporary read access to the object. */`

Source: api-reference-comments

---

## Parameters (A3)

A parameter description is a noun phrase describing the value, not a sentence about the parameter. Non-boolean parameters start with "The" or "A".

**Booleans take one of two patterns**, chosen by what the flag does:

| Flag means | Pattern | Example |
|------------|---------|---------|
| An action the call performs | `If true, … If false, …` | `If true, deletes the bucket and every object in it. If false, fails when the bucket still contains objects.` |
| A state the value carries | `True if …; false otherwise` | `True if the object is publicly readable; false otherwise.` |

**Optional parameters** say so and name the default at the end of the description: "Optional. Connection options such as the region and the request timeout. Defaults to the `us-east-1` region and a 30-second timeout." Where a generator already prints defaults from the signature, repeat them only where the reader can't see the signature — REST body tables and CLI help (convention).

**Units and ranges** belong in the description, not in the reader's head. Give the unit for every duration, size, rate, and price, and give the accepted range where one exists.

**Before → after:**

- Before: `@param timeout timeout`
- After: `@param timeout The time to wait for a response, in milliseconds. An integer from 1000 to 600000.`

- Before: `@param force force flag`
- After: `@param force If true, deletes the bucket even when it contains objects. If false, fails when the bucket isn't empty.`

- Before: `@param amount the amount @param currency currency (optional, default USD)`
- After: `@param amount The amount to charge, in the currency's smallest unit — 1099 means $10.99 for USD.` and `@param currency Optional. The three-letter ISO 4217 currency code. Defaults to usd.`

Source: api-reference-comments

---

## Return values and exceptions (A4)

Describe the return value with the same noun-phrase pattern as parameters: "The…" or "A…", brief, one sentence where possible. A boolean return uses `True if …; false otherwise`. A method returning nothing needs no return entry.

Exception wording depends on whether the tool prints the word "Throws" for you:

| Situation | Write | Renders as |
|-----------|-------|------------|
| Tool inserts "Throws" — JSDoc `@throws {E}`, Python `Raises: E:` | `If the bucket doesn't exist.` | Throws `NotFoundError` if the bucket doesn't exist |
| Nothing inserted — prose tables, hand-written reference pages | `Thrown when the bucket doesn't exist.` | as written |

Document every error type the method throws, one entry each — a caller writing a `catch` needs the full list, including errors raised by validation before any I/O happens.

Say what comes back when there's nothing to return. `null`, `undefined`, an empty array, and a thrown error are four different contracts, and a reader who guesses wrong ships a crash.

**Before → after:**

- Before: `@returns the objects @throws error if something goes wrong`
- After: `@returns The objects whose keys start with the prefix, sorted by key. Returns an empty array when no object matches.` plus `@throws {NotFoundError} If the bucket doesn't exist.` and `@throws {PermissionDeniedError} If the credentials can't list the bucket.`

- Before: `@returns boolean`
- After: `@returns True if the connection is open; false otherwise.`

Source: api-reference-comments

---

## Deprecations (A5)

A deprecated element names its replacement in the first sentence, because the reader is looking for what to call instead. Add the removal version or date, and one line of migration guidance concrete enough to apply without opening another page.

**Before → after:**

- Before: `/** @deprecated Deprecated. Do not use. */`
- After:

```
/**
 * Deprecated: use `createSignedUrl` instead, which returns a URL that expires.
 * Removed in v4.0.0 (2027-01-15). Replace `getObjectUrl(key)` with
 * `createSignedUrl(key, { expiresInSec: 3600 })`.
 *
 * Gets a permanent public URL for an object.
 */
```

Keep the original description below the deprecation note — readers still maintaining old code need it. Where a replacement doesn't exist, say what the reader does instead ("Store the object in a public bucket and read `object.publicUrl`.") rather than leaving them to guess.

Source: api-reference-comments

---

## Complete example: `StorageClient`

**Before** — typical weak comments: names restated, a flag undocumented, a thrown error invisible, a deprecation with no replacement.

```ts
/** Storage client */
export declare class StorageClient {
  /** @param bucket bucket @param options options */
  constructor(bucket: string, options?: ClientOptions);

  /** connected? */
  get isConnected(): boolean;

  /** region */
  get region(): string;

  /** This method deletes a bucket. @param force force flag */
  deleteBucket(force: boolean): Promise<boolean>;

  /** Uploads an object. */
  upload(key: string, body: Uint8Array): Promise<ObjectMetadata>;

  /** Deprecated, don't use. */
  getObjectUrl(key: string): string;
}
```

**After** — every rule in this file applied at once.

```ts
/**
 * Reads and writes objects in a single storage bucket.
 *
 * A client opens one connection per instance and reuses it for every request.
 * Create one client per bucket and share it across requests; the client is
 * safe for concurrent use.
 */
export declare class StorageClient {
  /**
   * Creates a client for the given bucket.
   *
   * @param bucket The name of an existing bucket in the caller's project.
   * @param options Optional. Connection options such as the region and the
   *     request timeout. Defaults to the `us-east-1` region and a 30-second
   *     timeout.
   */
  constructor(bucket: string, options?: ClientOptions);

  /**
   * Checks whether the client holds an open connection to the bucket.
   *
   * @returns True if the connection is open; false otherwise.
   */
  get isConnected(): boolean;

  /**
   * Gets the region that stores the bucket.
   *
   * @returns The region code, for example `us-east-1`.
   * @throws {NotConnectedError} If the client hasn't connected yet. Call
   *     `connect` first.
   */
  get region(): string;

  /**
   * Deletes the bucket.
   *
   * @param force If true, deletes the bucket and every object in it. If false,
   *     fails when the bucket still contains objects.
   * @returns True if the bucket was deleted; false if it didn't exist.
   */
  deleteBucket(force: boolean): Promise<boolean>;

  /**
   * Uploads an object and returns its stored metadata.
   *
   * The upload replaces any object with the same key. To make the write
   * conditional, pass a generation to `uploadIfGenerationMatch`.
   *
   * @param key The object key, up to 1024 bytes of UTF-8.
   * @param body The object contents.
   * @returns The metadata of the stored object, including its generation
   *     number and ETag.
   * @throws {QuotaExceededError} If the upload would exceed the project's
   *     storage quota.
   * @throws {NotConnectedError} If the client hasn't connected yet.
   */
  upload(key: string, body: Uint8Array): Promise<ObjectMetadata>;

  /**
   * Deprecated: use `createSignedUrl` instead, which returns a URL that
   * expires. Removed in v4.0.0 (2027-01-15). Replace `getObjectUrl(key)` with
   * `createSignedUrl(key, { expiresInSec: 3600 })`.
   *
   * Gets a permanent public URL for an object.
   *
   * @param key The object key.
   * @returns The public URL of the object.
   * @deprecated Use `createSignedUrl` instead.
   */
  getObjectUrl(key: string): string;
}
```

The same `upload` entry as a Go doc comment, in **Go Doc Comments** form. Go has no `@param` tags: the comment opens with the identifier ("Upload uploads…" rather than "Uploads…"), then covers parameters, units, errors, and concurrency in prose, with `[Name]` links:

```go
// Upload uploads body under key and returns the stored object's metadata,
// including its generation number and ETag. It replaces any object with the
// same key; to make the write conditional, use [Client.UploadIfGenerationMatch].
//
// The key is up to 1024 bytes of UTF-8. Upload returns [ErrQuotaExceeded] if
// the upload would exceed the project's storage quota, and [ErrNotConnected]
// if the client hasn't connected yet; test for them with [errors.Is]. It stops
// and returns ctx.Err() when ctx is canceled, and is safe for concurrent use.
func (c *Client) Upload(ctx context.Context, key string, body io.Reader) (ObjectMetadata, error)
```

Source: api-reference-comments; Go Doc Comments (go.dev/doc/comment)

---

## Reference voice (A7)

Reference entries describe; guides instruct. The difference is grammatical, and mixing the two inside one reference set makes entries read as inconsistent even when every fact is right.

| Surface | Voice | Example |
|---------|-------|---------|
| Reference entry | Third-person present, descriptive | `Creates a signed URL that expires after the given interval.` |
| Guide, tutorial, procedure step | Second person, imperative | `Create a signed URL, then send it to the browser.` |
| Usage note inside an entry | Second person, addressed to the caller | Call `connect` before you read `region`. |

Second person still earns its place in an entry's usage notes and constraints — the lines telling a caller what to do about the behavior just described. Keep it out of the summary line and the parameter, return, and exception descriptions.

Hold one tense across every entry in the set; a reference where some methods "return" and others "will return" reads as two documents merged. Behavior described in the present is true whenever the reader arrives.

**Before → after:**

- Before: `Use this method to fetch the invoice. It will return the invoice object.`
- After: `Gets the invoice for the given ID. Returns the invoice, including its line items.`

Voice rules for the surrounding prose — second person, active voice, present tense, no filler — are V1–V10 in [voice-and-words.md](voice-and-words.md).

Source: reference-verbs

---

## CLI help text (A6, convention)

Google has no page on `--help` output, so this section is **(convention)**, built on the guide's command-line syntax notation (P9, [procedures-and-code.md](procedures-and-code.md)) — `[optional]`, `{a|b}`, `...` — and its placeholder rules (P8, same file).

A complete `--help` screen carries seven parts in this order: usage line, one-line synopsis, description paragraph, positional arguments, flags with placeholder-style values and defaults, examples, and exit codes.

**Before:**

```
$ shipit deploy --help
Usage: shipit deploy

Deploys stuff. Simply pass your key and it will deploy your app really fast.

Options:
  --key         your api key (e.g. YOUR_API_KEY)
  --env         environment
  --dry-run     dry run
  --help        help
```

**After:**

```
$ shipit deploy --help
Usage: shipit deploy [OPTIONS] SERVICE_NAME

Deploys a service to an environment and waits for it to become healthy.

The command builds the image, uploads it, and replaces one replica at a time.
The rollout stops at the first replica that fails its health check.

Arguments:
  SERVICE_NAME           The name of the service to deploy, as listed by
                         `shipit services list`.

Options:
  --key API_KEY          The API key used to authenticate. Defaults to the
                         value of SHIPIT_API_KEY.
  --env {staging|prod}   The target environment. Default: staging.
  --replicas COUNT       The number of replicas to run. An integer from 1
                         to 50. Default: 3.
  --dry-run              If set, prints the deployment plan and exits
                         without changing anything.
  -h, --help             Prints this help text and exits.

Examples:
  Deploy the checkout service to staging:
    shipit deploy checkout

  Preview a production rollout with five replicas:
    shipit deploy --env prod --replicas 5 --dry-run checkout

Exit codes:
  0  The deployment succeeded.
  1  The deployment failed and was rolled back.
  2  An argument or flag was invalid.
  3  The API key was missing or rejected.
```

What the rewrite fixed: the usage line shows the optional and positional parts; the synopsis states the outcome instead of selling it; every flag names its value in placeholder caps and its default; the boolean flag uses the `If set, …` action pattern from A3; exit codes let a script branch on the result.

Source: convention (no Google `--help` page); code-syntax for the usage-line notation

---

## REST/HTTP reference pages (convention)

Google has no REST reference page type either, so the section list below is **(convention)**. The wording inside each cell is not: parameter descriptions follow A3, error descriptions follow A4.

A minimum endpoint page carries six parts: method and path as the heading, a one-sentence purpose, parameter tables split by location, the response with an example body, and an error table.

### POST /v1/buckets/{bucketId}/objects

Uploads an object to a bucket and returns its stored metadata.

The following table lists the path parameters:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `bucketId` | string | Yes | The ID of the bucket that receives the object. |

The following table lists the query parameters:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `ifGenerationMatch` | integer | No | The generation the object must currently have for the write to succeed. Omit to overwrite any generation. |

The following table lists the body parameters:

| Name | Type | Required | Description |
|------|------|----------|-------------|
| `key` | string | Yes | The object key, up to 1024 bytes of UTF-8. |
| `contentType` | string | No | The MIME type stored with the object. Default: `application/octet-stream`. |
| `public` | boolean | No | If true, grants read access to anyone with the URL. If false, restricts access to the bucket's ACL. Default: false. |

A successful request returns `201 Created` and the object's metadata:

```json
{
  "key": "invoices/2026-08.pdf",
  "generation": 1724947200000001,
  "sizeBytes": 48213,
  "etag": "d41d8cd98f00b204e9800998ecf8427e",
  "createdAt": "2026-08-29T10:00:00Z"
}
```

The following table lists the errors this endpoint returns:

| Status | Code | Description |
|--------|------|-------------|
| 404 | `bucket_not_found` | Returned when no bucket has the given ID, or the credentials can't see it. |
| 409 | `generation_mismatch` | Returned when `ifGenerationMatch` doesn't equal the object's current generation. |
| 413 | `object_too_large` | Returned when the body exceeds the 5 TiB per-object limit. |
| 429 | `quota_exceeded` | Returned when the project exceeds its write rate. Retry after the interval in the `Retry-After` header. |

Give every endpoint the same section order and the same tables, including single-row ones. A reader scanning six endpoints reads position, not prose: a page that drops its query-parameter table reads as an endpoint that takes none.

Source: convention (no Google REST reference page); api-reference-comments for the parameter and error wording
