# Hosted documentation attributions

This page covers third-party material delivered by or referenced from the Hero
documentation site. Third-party components retain their own licenses.

| Component | Use | License / attribution |
|---|---|---|
| Material for MkDocs | Generated site theme, CSS, JavaScript, and icons | MIT; copyright Martin Donath and contributors |
| MkDocs | Static-site generator | BSD-2-Clause |
| Lunr | English search engine bundled in the generated worker | MIT; copyright Oliver Nightingale |
| TinySegmenter | Upstream language-search support shipped by Material | BSD-style license; copyright Taku Kudo |
| Font Awesome GitHub icon | Repository icon emitted by the theme | CC BY 4.0; Fonticons, Inc. |
| Pictogrammers Material Design Icons | Theme navigation and palette icons | Apache-2.0 |
| Inter and JetBrains Mono | Fonts fetched from Google Fonts | SIL Open Font License 1.1 |

Build dependencies are pinned in `requirements-docs.txt`. The English deploy
removes Material's unused Lunr language directory after the strict build. That
removal excludes the copied MPL-1.1 Lunr Languages files and LGPL-3.0 Wordcut
bundle from the deployed artifact; the sanitizer fails if those filenames
remain. Source maps are also removed so unused copied source is not published.

The generated English search worker retains Lunr itself under MIT. The site
does not claim that any third-party component is relicensed by Hero.
