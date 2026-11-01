import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const read = (path) => readFileSync(new URL(path, import.meta.url), 'utf8');
const contract = JSON.parse(read('../../../design-system.json'));
const css = read('./styles/tokens.css');

const normalize = (value) => value.replace(/\s+/g, ' ').trim();

function flattenTokens(node, path = '') {
  const tokens = new Map();
  for (const [key, value] of Object.entries(node)) {
    if (key.startsWith('$')) continue;
    const next = path ? `${path}.${key}` : key;
    if (value && typeof value === 'object' && '$value' in value) {
      tokens.set(next, value);
    } else if (value && typeof value === 'object') {
      for (const [childPath, child] of flattenTokens(value, next)) tokens.set(childPath, child);
    }
  }
  return tokens;
}

const tokens = flattenTokens(contract.tokens);
const cssVars = new Map();
for (const [, name, value] of css.matchAll(/(--tk-[a-z0-9-]+)\s*:\s*([^;]+);/g)) {
  cssVars.set(name, normalize(value));
}

function resolveCss(value) {
  const match = value.match(/^var\((--tk-[a-z0-9-]+)\)$/);
  if (!match) return value;
  assert.ok(cssVars.has(match[1]), `referencia CSS sin variable: ${match[1]}`);
  return resolveCss(cssVars.get(match[1]));
}

function resolveToken(value) {
  if (typeof value !== 'string') return value;
  const match = value.match(/^\{([^}]+)\}$/);
  if (!match) return value;
  const target = tokens.get(match[1]);
  assert.ok(target, `referencia DTCG sin token: ${match[1]}`);
  return resolveToken(target.$value);
}

test('cada token del contrato coincide con su variable de tokens.css', () => {
  for (const [path, token] of tokens) {
    const cssVar = token.$extensions?.['tasks-ai']?.cssVar;
    assert.ok(cssVar, `${path}: falta $extensions.tasks-ai.cssVar`);
    assert.ok(cssVars.has(cssVar), `${path}: ${cssVar} no existe en tokens.css`);
    assert.equal(
      resolveCss(cssVars.get(cssVar)),
      normalize(resolveToken(token.$value)),
      `${path}: el contrato y ${cssVar} no coinciden`
    );
  }
});

test('tokens.css no declara variables que falten en el contrato', () => {
  const declared = new Set([...tokens.values()].map((token) => token.$extensions['tasks-ai'].cssVar));
  for (const name of cssVars.keys()) {
    assert.ok(declared.has(name), `${name} no está declarada en design-system.json`);
  }
});

test('el tipo declarado corresponde a la forma del valor', () => {
  for (const [path, token] of tokens) {
    const value = String(resolveToken(token.$value));
    if (token.$type === 'color') {
      assert.match(value, /^(#[0-9a-f]{3,8}|rgba?\()/i, `${path}: no es un color`);
    } else if (token.$type === 'dimension') {
      assert.match(value, /^-?\d+(\.\d+)?(px|rem|em|%)$/, `${path}: no es una dimensión`);
    } else {
      assert.equal(token.$type, 'fontFamily', `${path}: tipo inesperado`);
    }
  }
});
