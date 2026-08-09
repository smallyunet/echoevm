export interface SolidityFunctionDeclaration {
  name: string;
  offset: number;
}

export function scanSolidityFunctions(source: string): SolidityFunctionDeclaration[] {
  const searchable = maskCommentsAndStrings(source);
  const pattern = /\bfunction\s+([A-Za-z_$][A-Za-z0-9_$]*)\s*\(/gu;
  const declarations: SolidityFunctionDeclaration[] = [];
  for (const match of searchable.matchAll(pattern)) {
    const name = match[1];
    const relative = match[0].indexOf(name);
    if (name && match.index !== undefined && relative >= 0) {
      declarations.push({ name, offset: match.index + relative });
    }
  }
  return declarations;
}

function maskCommentsAndStrings(source: string): string {
  const characters = [...source];
  let state: "code" | "line-comment" | "block-comment" | "single-string" | "double-string" = "code";
  for (let index = 0; index < characters.length; index++) {
    const current = characters[index];
    const next = characters[index + 1];
    if (state === "code") {
      if (current === "/" && next === "/") {
        characters[index] = characters[index + 1] = " ";
        index++;
        state = "line-comment";
      } else if (current === "/" && next === "*") {
        characters[index] = characters[index + 1] = " ";
        index++;
        state = "block-comment";
      } else if (current === "'") {
        characters[index] = " ";
        state = "single-string";
      } else if (current === '"') {
        characters[index] = " ";
        state = "double-string";
      }
      continue;
    }
    if (current === "\n" && state === "line-comment") {
      state = "code";
      continue;
    }
    if (state === "block-comment" && current === "*" && next === "/") {
      characters[index] = characters[index + 1] = " ";
      index++;
      state = "code";
      continue;
    }
    if ((state === "single-string" || state === "double-string") && current === "\\") {
      characters[index] = " ";
      if (index + 1 < characters.length && characters[index + 1] !== "\n") {
        characters[index + 1] = " ";
        index++;
      }
      continue;
    }
    if ((state === "single-string" && current === "'") || (state === "double-string" && current === '"')) {
      characters[index] = " ";
      state = "code";
      continue;
    }
    if (current !== "\n") {
      characters[index] = " ";
    }
  }
  return characters.join("");
}
