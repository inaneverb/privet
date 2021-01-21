<p align="center">
  <img alt="Privet" height="125" src="https://raw.githubusercontent.com/qioalice/privet/master/.github/logo.svg">
  <br>
</p>
<p align="center">
Privet is an another one Golang i18n (internationalization) package, that makes you stop to hard code displayed language phrases 
and move out all translates to the separated locale files.
You can load them once at the your service initialization and then get required language's phrases at the runtime whenever you need.
And thanks to avoid of html/template or fmt.Sprintf, the interpolation of translated phrases so fast.
<br>
Just try!

---

# Quick start

```go
package main

import (
	"fmt"
	"github.com/qioalice/privet/v2"
)

func main() {

	const (
		en_US = `
__metadata__:
  locale: en_US
a:
  b: "Hello, {{name}}!"
`
		zh_CN = `
__metadata__:
  locale: zh_CN
a:
  b: "你好, {{name}}!"
`
	)

	privet.Source([]byte(en_US), []byte(zh_CN)).LogAsFatal()
	privet.Load().LogAsFatal()

	privet.LC("en_US").MarkAsDefault()

	enPhrase := privet.Tr("en_US", "a/b", privet.Args{
		"name": "Frank",
	})
	zhPhrase := privet.Tr("zh_CN", "a/b", privet.Args{
		"name": "Dave",
	})
	unexistedLocalePhrase := privet.Tr("ru_RU", "a/b", nil)
	
	fmt.Println(enPhrase) // "Hello, Frank!" 
	fmt.Println(zhPhrase) // "你好, Dave!" 
	fmt.Println(unexistedLocalePhrase) // "Hello, {{name}}!" (en_US is default locale, args are not presented).
}
```

# Loading locales

First you need to know that the mechanism of loading (or re-loading) sources of locales
contains two parts:
* Declaring the **NEW sources** of locales, analyse them, calculates MD5 hash sums, etc
* Parse and **load** them, recognize locales, overwrite, etc - for all prepared new sources

And there is a two functions to do this.

## Specify the sources

There is a function `Source()`, and you may use it to specify:
- A filepath to the source of locale(s),
- A path to the directory contains files that are source(s) of locale(s)...
- ... or also contains a directories that contains a directories, that ...
- A RAW data (content) of source(s) of locale(s),
- An array of any of the above.

Any other variants of arguments are prohibited and will return an error,
not changing the sources, already prepared for being loaded.

```go
// ./locales
//    |- ru_RU
//         |- locale_file_1_1.toml
//         |- locale_file_1_2.yaml
//         |- unsupported_file.html
//    |- en_US
//         |- locale_file_2_1.json

privet.Source("./locales") 

// This will scan all of ["locale_file_1_1.toml", "locale_file_1_2.toml", "locale_file_2_1.toml"]
// but ignores "unsupported_file.html".
```

## Choose the format

No matter, you want to load locales from files or directly specify its RAW data, you need to know what format is allowed. The answer is:

- **TOML v1.0-rc3**: https://toml.io/en/ , https://en.wikipedia.org/wiki/TOML
- **YAML v1.2**: https://yaml.org/ , https://en.wikipedia.org/wiki/YAML
- **JSON**: https://www.json.org/json-en.html , https://en.wikipedia.org/wiki/JSON

<p>
<sub>
Technically, JSON is supported because it's a subset of YAML v1.2. Thus, parsing JSON using YAML decoder is safe and that's exactly how JSON is supported. Moreover, you can use ANY other markup language that is a subset of any already supported languages.
</sub>
</p>

So. All your sources you want to count must be encoded using any of that format. You can use all of them at the same time if you want.

## Name your locale

You need to specify locale name. Of course, how would you recognize what locale your file contains if it's unnamed? But before we will proceed, read the limitations. And do not cross the line.

### Requirements and limitations
- One source *MUST* contain *ONLY ONE* locale name. No matter where. Counts everywhere it could be. I mean, there is no "priority" of locale name. If your source contains two or more locale names - its an error.
- Locale name *MUST* have the following format: `en_US`. [This is LCID](https://en.wikipedia.org/wiki/Locale_(computer_software)). And you must use *EXACTLY* that format. Neither just locale name w/o country code like "en" nor any other delimeter in LCID. If your app uses another locale's ID format, just write a translator.

It's not that hard, right? Now see, what will you get.

### Where locale name could be?

The short answer is: "At the any part of source's header or inside source's content".
We can say that there is a two big categories of sources:

- A files. They has "a header". It's a file's path, metadata, etc.
- A content. It DOES NOT has "a header". Indeed, what could it be?

So, let's start from the easy way.
You can specify locale **inside your file's content** (or inside RAW data). I won't show you examples for all supported locale formats. I will show you JSON and you may found examples for TOML, YAML at the `/examples` directory.

```json
{
    "__metadata__": {
        "locale": "en_US"
    }
}
```

<p>
<sub>
Keys <code>__metadata__</code> and <code>locale</code> are case insensitive. That means you may capitalize it, mixing or anything else. Moreover, there are few keys to specify locale's name. Its: <code>locale</code>, <code>localename</code>, <code>locale_name</code>, <code>name</code>. Case insensitive allows you to use keys in PascalCase or camelCase format.
<br>
There is only one variant of metadata key, but also case insensitive.
</sub>
</p>

But there is another way to specify locale name if you don't want to put locale name directly to your file's content. Maybe you think this is kinda ugly.

So, you may **specify locale name inside any part of filepath**. Keep in mind, that only one locale name is allowed. So, its either filepath, either metadata section in content.
Speaking about filepath, **locale name could be in a directory name, subdirectory name, file name, or even be a part of any of that**! Take a look:

<ul>
<li><code>./locales/<b>en_US</b>/content/file1.json</code> - Locale name from a directory</li>
<li><code>/etc/app/locales/<b>en_US.json</b></code> - Locale name from a filename</li>
<li><code>~/.app/locales/<b>ru_RU_part1</b>/file1.json</code> - Locale name inside directory's name</li>
</ul>

So, if you places locale's name inside some string either directory name or filename, it must be wrapped by delimeters to be treated as locale name. Allowed delimeters are: "-_. ": hyphen, underscore, dot and space. Dot allows you to combine it in filename more "natural" way. Like `section1.en_US.json`, `text.en_US.text2.json`, etc.

## Do locales loading

Until you do not call `Load()`, locales counted by `Source()` are not loaded.
Exactly `Load()` changes all internal structures, compares MD5 hashsums of all sourced locales, finding the sames to avoid multitimes loading of the same source and loads all of them.


# Default locale

In your code I assume you will write things like:
```go
func translate(localeName string) string {
    return privet.Tr(localeName, "A/B/C", nil)
}
```

But what if requested locale does not exist? You will get a string like:
`i18nErr: LocaleIsNil. Key: A/B/C` instead of desired language phrase. Maybe would you prefer to use some locale as default to handle all that cases? No problem.

```go
privet.LC("en_US").MarkAsDefault() // will set en_US locale as default
```

# Translation errors

Sometimes function `Tr()` or method `Locale.Tr()` may face an unforeseen situation. One of that you already seen in the section above while we talk about default locales.
Let's talk about the rest.

First of all. `Tr()` (or `Locale.Tr()`) returns ONLY a string. It's a function signature. Normally it returns a language phrase, but sometimes things may changed and error is occurred.
The format of error string is:

`i18nErr: <ErrorMessage>. Key: <TranslationKey>`

As shown above, `<TranslationKey>` is YOUR translation key. The key by what did you want to get a translation phrase but something went wrong.

Variants of `<ErrorMessage>`:

| Class | Meaning |
| --- | --- |
| `TranslationNotFound` | Translation not found for requested locale and translation key. <br>Maybe avoided by marking any locale as default to use it instead, if requested locale is not exist. But if locale is exist and just do not contains phrase for desired key, you will still get this error. |
| `LocaleIsNil` | Locale not found.<br>Requested locale not found or maybe you manually instantiate `Locale` class and trying to interact with?|
| `TranslationKeyIsEmpty` | Your translation key is empty.|
| `TranslationKeyIsIncorrect` | Your translation key is malformed and incorrect.<br>E.g: Leading or trailing slash; contains just slash, nothing more; etc.|

# FAQ

Q: **Thread-safety?**<br>
A: Yes, partially. If you loading all your locales just once and then just getting language phrases by translation keys - all is good. There is even no "thread-safety" or "thread-unsafety" at all. No data race, no that sentences.<br>

Q: **Thread-safety and re-loading?**<br>
A: It keeps you away from panics or UB. But you need to write your own syncers and lockers. For example, you will get `LocaleIsNil` translation error until locales loaded successfully.<br>

Q: **Multiple `Source()` calls?**<br>
A: Yes, all of that sources will be counted. But you cannot call `Source()` while its already called in another goroutine. It will return an error.<br>

Q: **Multiple `Load()` calls?**<br>
A: If you do not specify any source before these calls, it returns an error. Technically it means, that you want to load locales w/o any specified source. If you call `Load()` when another goroutines also executes it, error is returned.<br>

Q: **Locales were loaded, I tried to reload but get an error. What happens next?**<br>
A: If some locales were load successfully before (you had at least one successful `Load()` call at all), these locales will be used. You still may get translations. But if there was no successfully loaded locales, you will get an error.

Q: **More than one translation entry points?**<br>
A: Yes. That's the reason `Client` type is exposed. Typically all package level functions like `Load()`, `Source()`, `Tr()`, `LC()`, and others are aliases to default client's methods. You may instantiate `Client` object and use it instead of package's functions. That type is ready-to-use after instantiating just like you using a package. Call Source(), call Load(), then get translations. You know that already.<br>

Q: **How handle `Source()` and `Load()` errors?**<br>
A: These functions returns `*ekaerr.Error` object, and you may anaylse that, throw, ignore or log. It's highly integrated with `ekaerr` and `ekalog` package from `ekago` framework library. [Read more about Ekago.](https://github.com/qioalice/ekago)

# To do

- Expose `Client`'s config variables (using public setters/getters) to allow user to specify how edge cases must be resolved.
- Comment all entities that is not commented yet.
- Improve interpolation and fully reject to use `fmt.Sprintf()`

# Contribution

PRs and issues are welcome. If you want to help I would appretiate it.

---

<p align="center">
  <sub>
    The logo is <a href="https://ui8.net/lukian025/products/20-space-explorer-icons">the purchased item from the UI8 resource</a>
    and <a href="https://ui8.net/licensing">all rights reserved</a>.
    You cannot use that logo in your own purposes w/o purchasing it for yourself.
  </sub>
</p>
