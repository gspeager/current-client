export default {
  extends: ['stylelint-config-standard-scss'],
  rules: {
    // Prettier owns blank lines.
    'at-rule-empty-line-before': null,
    'declaration-empty-line-before': null,
    'custom-property-empty-line-before': null,
    'scss/double-slash-comment-empty-line-before': null,
    // max-width reads in every WebKit the app runs on, Linux's older WebKitGTK included.
    'media-feature-range-notation': 'prefix',
    // WebKit still needs -webkit-user-select and -webkit-text-size-adjust.
    'property-no-vendor-prefix': null,
    'value-no-vendor-prefix': null,
    // Colors are written as design-system.md lists them, which the theme tests read.
    'color-hex-length': null,
    'color-function-notation': null,
    'color-function-alias-notation': null,
    'alpha-value-notation': null,
    // CLAUDE.md: lowercase, words joined by single hyphens, starting with a letter.
    'selector-class-pattern': [
      '^[a-z][a-z0-9]*(-[a-z0-9]+)*$',
      { message: 'Class names are lowercase words joined by single hyphens' },
    ],
    // Tokens over literals: design-system.md's scales live in styles/tokens.
    'declaration-property-value-allowed-list': {
      'z-index': ['/^var\\(--z-/', 'auto', '0'],
    },
    'declaration-property-value-disallowed-list': {
      '/^(padding|margin|gap|row-gap|column-gap)/': ['/(^|[\\s(])(2|3|4|5|6|8|12|16|24)px/'],
      '/^border(-[a-z]+)*-radius$/': ['/(^|\\s)(2|3|4|6|8|12)px/'],
    },
  },
}
