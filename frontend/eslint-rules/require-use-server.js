/**
 * Enforces the Server Action module conventions used in src/actions:
 *
 * 1. The file must start with the 'use server' directive, so every export is
 *    compiled as a Server Action reference instead of being bundled into the
 *    client.
 * 2. Every runtime export must be an async function. Next.js rejects anything
 *    else in a 'use server' module at build time; this surfaces it in the editor.
 */
function hasUseServerDirective(program) {
  for (const statement of program.body) {
    if (statement.type !== "ExpressionStatement" || typeof statement.directive !== "string") {
      return false;
    }
    if (statement.directive === "use server") {
      return true;
    }
  }
  return false;
}

function isAsyncFunction(node) {
  return (
    !!node &&
    (node.type === "FunctionDeclaration" ||
      node.type === "FunctionExpression" ||
      node.type === "ArrowFunctionExpression") &&
    node.async === true
  );
}

/** @type {import("eslint").Rule.RuleModule} */
module.exports = {
  meta: {
    type: "problem",
    docs: {
      description: "Require 'use server' and async-only exports in Server Action modules",
    },
    schema: [],
    messages: {
      missingDirective:
        "Server Action modules must start with the 'use server' directive. See docs/SERVER_ACTIONS.md.",
      nonAsyncExport:
        "Only async functions can be exported from a 'use server' module. Move other exports to a separate file.",
    },
  },
  create(context) {
    return {
      Program(program) {
        if (!hasUseServerDirective(program)) {
          context.report({ node: program.body[0] ?? program, messageId: "missingDirective" });
        }
      },
      ExportNamedDeclaration(node) {
        if (node.exportKind === "type") {
          return;
        }
        const { declaration } = node;
        if (!declaration) {
          // `export { a }` / re-exports can't be verified locally; require
          // explicit async declarations instead.
          context.report({ node, messageId: "nonAsyncExport" });
          return;
        }
        if (
          declaration.type === "TSTypeAliasDeclaration" ||
          declaration.type === "TSInterfaceDeclaration" ||
          declaration.declare
        ) {
          return;
        }
        if (declaration.type === "FunctionDeclaration") {
          if (!declaration.async) {
            context.report({ node: declaration, messageId: "nonAsyncExport" });
          }
          return;
        }
        if (declaration.type === "VariableDeclaration") {
          for (const declarator of declaration.declarations) {
            if (!isAsyncFunction(declarator.init)) {
              context.report({ node: declarator, messageId: "nonAsyncExport" });
            }
          }
          return;
        }
        context.report({ node: declaration, messageId: "nonAsyncExport" });
      },
      ExportDefaultDeclaration(node) {
        if (!isAsyncFunction(node.declaration)) {
          context.report({ node, messageId: "nonAsyncExport" });
        }
      },
      ExportAllDeclaration(node) {
        if (node.exportKind !== "type") {
          context.report({ node, messageId: "nonAsyncExport" });
        }
      },
    };
  },
};
