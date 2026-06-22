/**
 * @typedef {Object} Directive
 * @property {string} name
 * @property {string[]} [args]
 * @property {number} line
 * @property {Directive[]} [block]
 */

/**
 * @typedef {Object} FlowStep
 * @property {string} name
 * @property {boolean} known
 */

/**
 * @typedef {Object} ServerBlock
 * @property {string[]} keys
 * @property {number} line
 * @property {Directive[]} directives
 * @property {FlowStep[]} [flow]
 */

/**
 * @typedef {Object} Corefile
 * @property {ServerBlock[]} serverBlocks
 */

/**
 * @typedef {Object} Diagnostic
 * @property {"error"|"warning"|"info"} severity
 * @property {string} message
 * @property {number} line
 */

/**
 * @typedef {Object} Result
 * @property {Corefile|null} corefile
 * @property {Diagnostic[]} diagnostics
 */

export {}
