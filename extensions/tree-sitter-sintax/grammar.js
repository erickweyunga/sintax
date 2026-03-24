module.exports = grammar({
  name: "sintax",

  extras: ($) => [/\s/, $.comment, $.block_comment],

  word: ($) => $.identifier,

  rules: {
    source_file: ($) => repeat($._statement),

    _statement: ($) =>
      choice(
        $.use_statement,
        $.function_definition,
        $.if_statement,
        $.while_statement,
        $.for_statement,
        $.match_statement,
        $.return_statement,
        $.print_statement,
        $.typed_assignment,
        $.assignment,
        $.compound_assignment,
        $.index_assignment,
        $.expression_statement
      ),

    // use "std/math"
    use_statement: ($) =>
      seq("use", field("path", choice($.string, $.single_string))),

    // fn (params) [return_type] name:
    function_definition: ($) =>
      seq(
        "fn",
        "(",
        optional($.parameter_list),
        ")",
        optional(field("return_type", $.type)),
        field("name", $.identifier),
        ":",
        field("body", $.block)
      ),

    parameter_list: ($) => seq($.parameter, repeat(seq(",", $.parameter))),

    parameter: ($) =>
      choice(
        $.typed_parameter,
        field("name", $.identifier)
      ),

    typed_parameter: ($) =>
      seq(field("type", $.type), field("name", $.identifier)),

    type: ($) => choice("num", "str", "bool", "list", "dict"),

    block: ($) => prec.right(repeat1($._statement)),

    // if condition:
    if_statement: ($) =>
      seq(
        "if",
        field("condition", $._expression),
        ":",
        field("body", $.block),
        optional($.else_clause)
      ),

    else_clause: ($) => seq("else", ":", field("body", $.block)),

    // while condition:
    while_statement: ($) =>
      seq(
        "while",
        field("condition", $._expression),
        ":",
        field("body", $.block)
      ),

    // for var in iterable:
    for_statement: ($) =>
      seq(
        "for",
        field("variable", $.identifier),
        "in",
        field("iterable", $._expression),
        ":",
        field("body", $.block)
      ),

    // match expr:
    match_statement: ($) =>
      seq(
        "match",
        field("value", $._expression),
        ":",
        repeat($.case_clause),
        optional($.default_clause)
      ),

    case_clause: ($) =>
      seq("case", field("value", $._expression), ":", field("body", $.block)),

    default_clause: ($) => seq("_", ":", field("body", $.block)),

    // return expr
    return_statement: ($) => seq("return", $._expression),

    // >> expr
    print_statement: ($) => seq(">>", $._expression),

    // type name = expr
    typed_assignment: ($) =>
      seq(field("type", $.type), field("name", $.identifier), "=", field("value", $._expression)),

    // name = expr
    assignment: ($) =>
      seq(field("name", $.identifier), "=", field("value", $._expression)),

    // name += expr
    compound_assignment: ($) =>
      seq(
        field("name", $.identifier),
        field("operator", choice("+=", "-=", "*=", "/=")),
        field("value", $._expression)
      ),

    // name[idx] = expr
    index_assignment: ($) =>
      seq(
        field("name", $.identifier),
        repeat1($.index_access),
        "=",
        field("value", $._expression)
      ),

    expression_statement: ($) => $._expression,

    // Expression precedence (low to high)
    _expression: ($) => $.or_expression,

    or_expression: ($) =>
      prec.left(1, seq($._expression, "or", $._expression)),

    and_expression: ($) =>
      prec.left(2, seq($._expression, "and", $._expression)),

    comparison: ($) =>
      prec.left(
        3,
        seq(
          $._expression,
          field(
            "operator",
            choice("==", "!=", ">", "<", ">=", "<=", "in")
          ),
          $._expression
        )
      ),

    addition: ($) =>
      prec.left(4, seq($._expression, choice("+", "-"), $._expression)),

    multiplication: ($) =>
      prec.left(5, seq($._expression, choice("*", "/", "%"), $._expression)),

    power: ($) => prec.right(6, seq($._expression, "**", $._expression)),

    not_expression: ($) => prec(7, seq("not", $._expression)),

    negation: ($) => prec(7, seq("-", $._expression)),

    positive: ($) => prec(7, seq("+", $._expression)),

    // Primary expressions
    _primary: ($) =>
      choice(
        $.lambda,
        $.call_expression,
        $.method_call,
        $.index_expression,
        $.list_literal,
        $.dict_literal,
        $.number,
        $.string,
        $.single_string,
        $.identifier,
        $.parenthesized_expression,
        $.true,
        $.false,
        $.null
      ),

    // fn(params) -> expr
    lambda: ($) =>
      seq(
        "fn",
        "(",
        optional(seq($.identifier, repeat(seq(",", $.identifier)))),
        ")",
        "->",
        field("body", $._expression)
      ),

    // name(args)
    call_expression: ($) =>
      prec(8, seq(field("function", $.identifier), "(", optional($.argument_list), ")")),

    argument_list: ($) => seq($._expression, repeat(seq(",", $._expression))),

    // expr.method(args)
    method_call: ($) =>
      prec.left(
        9,
        seq(
          field("object", $._expression),
          ".",
          field("method", $.identifier),
          "(",
          optional($.argument_list),
          ")"
        )
      ),

    // expr[index]
    index_expression: ($) =>
      prec.left(9, seq($._expression, "[", $._expression, "]")),

    index_access: ($) => seq("[", $._expression, "]"),

    // [elem, ...]
    list_literal: ($) =>
      seq("[", optional(seq($._expression, repeat(seq(",", $._expression)))), "]"),

    // {"key": value, ...}
    dict_literal: ($) =>
      seq(
        "{",
        optional(seq($.dict_entry, repeat(seq(",", $.dict_entry)))),
        "}"
      ),

    dict_entry: ($) => seq(field("key", $._expression), ":", field("value", $._expression)),

    parenthesized_expression: ($) => seq("(", $._expression, ")"),

    // Literals
    number: ($) => /\d+(\.\d+)?/,

    string: ($) =>
      seq('"', repeat(choice($.string_content, $.escape_sequence, $.interpolation)), '"'),

    string_content: ($) => /[^"\\{]+/,

    escape_sequence: ($) => /\\[ntr0\\"]/,

    interpolation: ($) => seq("{", $.identifier, "}"),

    single_string: ($) => seq("'", optional(/[^']*/), "'"),

    identifier: ($) => /[a-zA-Z_]\w*/,

    true: ($) => "true",
    false: ($) => "false",
    null: ($) => "null",

    // Comments
    comment: ($) => seq("--", /.*/),

    block_comment: ($) => seq("--{", /[^}]*/, "}--"),
  },
});
