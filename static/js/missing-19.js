// @license magnet:?xt=urn:btih:87f119ba0b429ba17a44b4bffcab33165ebdacc0&dn=freebsd.txt BSD-2-Clause
/**
 * a DOM helper library.
 * "1 US$ = 18.5842 TR₺ · Oct 16, 2022, 20:52 UTC"
 */

// @ts-check

/// <reference lib="es2022" />
/// <reference lib="dom" />

/**
 * @template TOptions
 * @callback Behavior
 * @param {ParentNode} subtree
 * @param {Partial<TOptions>} [options]
 * @returns {void}
 */

/**
 * @template TOptions
 * @callback BehaviorInit
 * @param {Element} element
 * @param {BehaviorContext<TOptions>} context
 * @returns void
 */

/**
 * @template TOptions
 * @typedef {object} BehaviorContext
 * @property {Root} root
 * @property {Partial<TOptions>} options
 */

/**
 * @typedef {Document | ShadowRoot} Root
 */

/**
 * @typedef {<T>(...args: [..._: unknown[], last: T]) => T} Logger
 */

/**
 * Creates a logging function.
 * The {@link scope} will be prepended to each message logged.
 *
 * We usually use `ilog` as a name for the resulting function.
 * It returns its last argument,
 * which makes it easier to log intermediate values in an expression:
 *
 *     const x = a + ilog("b:", b); // x = a + b
 *
 * @param {string} scope The name of the component/module/etc. that will use this logger.
 * @returns {Logger} The `ilog` function.
 */
export function makelogger(scope) {
  /**
   * @template T
   */
  return (...args) => {
    console.log("%c%s", "color:green", scope, ...args);
    return /** @type {T} */ (args.at(-1));
  };
}

/**
 * Converts camelCase to kebab-case.
 * @param {string} s
 * @returns {string}
 */
function camelToKebab(s) {
  return s.replace(/[A-Z]/g, (part) => "-" + part.toLowerCase());
}

/**
 * Traverse the DOM forward or backward from a starting point
 * to find an element matching some selector.
 * @param {"next" | "previous"} direction
 * @param {ParentNode} root
 * @param {string} selector
 * @param {Element | null} current
 * @param {object} [options]
 * @param {boolean} [options.wrap]
 */
export function traverse(direction, root, selector, current, options = {}) {
  const { wrap = true } = options;
  const advance = /** @type {const} */ (`${direction}ElementSibling`);

  const wrapIt = () => {
    if (!wrap) return null;
    return direction === "next" ? $(root, selector) : $$(root, selector).at(-1);
  };

  if (!current) return wrapIt();

  let cursor = current;
  while (true) {
    while (cursor[advance] === null) {
      cursor = /** @type {HTMLElement} */ (cursor.parentElement);
      if (cursor === root) return wrapIt();
    }
    cursor = /** @type {HTMLElement} */ (cursor[advance]);
    const found = cursor.matches(selector) ? cursor : $(cursor, selector);
    if (found) return found;
  }
}

/**
 * @template {Element} TElement
 * @param {ParentNode} scope
 * @param {string} sel
 * @returns {TElement | null}
 */
export function $(scope, sel) {
  return scope.querySelector(sel);
}

/**
 * @template {Element} TElement
 * @param {ParentNode} scope
 * @param {string} sel
 * @returns {TElement[]}
 */
export function $$(scope, sel) {
  return Array.from(scope.querySelectorAll(sel));
}

/**
 * @typedef {object} EventListenerToken
 * @property {EventTarget} target
 * @property {string} type
 * @property {EventListener} listener
 * @property {object} options
 */

/**
 * @template T extends string
 * @callback Listener
 * @param {T extends keyof HTMLElementEventMap ? HTMLElementEventMap[T] : CustomEvent} event
 * @returns void
 */

/**
 * @template {string} TEventType
 * @param {EventTarget} target
 * @param {TEventType} type
 * @param {Listener<TEventType>} listener
 * @param {object} [options]
 * @param {Element} [options.addedBy]
 */
export function on(target, type, listener, options = {}) {
  /** @type {Listener<TEventType>} */
  const listenerWrapper = (event) => {
    if (options.addedBy && !options.addedBy.isConnected) {
      off({
        target,
        type: type,
        listener: /** @type {EventListener} */ (listenerWrapper),
        options,
      });
    }
    if ("logEvents19" in window) console.log(event);
    return listener(event);
  };
  target.addEventListener(
    type,
    /** @type {EventListener} */ (listenerWrapper),
    /** @type {AddEventListenerOptions} */ (options),
  );
  return { target, type: type, options, listener: listenerWrapper };
}

/**
 * @param {EventListenerToken} listenerToken
 */
export function off({ target, type, listener, options }) {
  return target.removeEventListener(type, listener, options);
}

/**
 * @param {string} mode
 * @param {Event} event
 */
export function halt(mode, event) {
  for (const token of mode.split(" ")) {
    if (token === "default") event.preventDefault();
    if (token === "bubbling") event.stopPropagation();
    if (token === "propagation") event.stopImmediatePropagation();
  }
}

/**
 * @template {Event} T
 * @param {string} mode
 * @param {(e: T) => void} listener
 * @returns {(e: T) => void}
 */
export function halts(mode, listener) {
  return (event) => {
    halt(mode, event);
    listener(event);
  };
}

/**
 * @param {EventTarget} el
 * @param {string} type
 * @param {unknown} [detail]
 * @param {CustomEventInit} [options]
 */
export function dispatch(el, type, detail, options) {
  return el.dispatchEvent(new CustomEvent(type, { ...options, detail }));
}

/**
 * @param {Element} el
 * @param {string | Record<string, unknown>} name
 * @param {unknown} value
 * @returns {string | null}
 */
export function attr(el, name, value) {
  if (typeof name === "object") {
    for (const key in name) el.setAttribute(camelToKebab(key), String(name[key]));
    return null;
  }

  const currentValue = el.getAttribute(name);
  if (value === undefined) return el.getAttribute(name);
  if (value === null) {
    el.removeAttribute(name);
    return currentValue;
  }

  el.setAttribute(name, String(value));
  return String(value);
}

/**
 * Generate a random ID, copied from nanoid.
 */
export function mkid() {
  return Array.from(
    { length: 21 },
    () =>
      "useandom-26T198340PX75pxJACKVERYMINDBUSHWOLF_GQZbfghjklqvwyzrict"[
        (Math.random() * 64) | 0
      ],
  ).join("");
}

/**
 * @param {Element} el
 */
export function identify(el) {
  if (el.id) return el.id;
  el.id = mkid();
  return el.id;
}

/**
 * @param {Node} node
 * @returns string
 */
export function stringifyNode(node) {
  const tmp = document.createElement("div");
  tmp.append(node);
  return tmp.innerHTML;
}

/**
 * @param {unknown} input
 * @returns {string}
 */
export function htmlescape(input) {
  if (input === null || input === undefined) return "";
  if (input instanceof Node) return stringifyNode(input);
  return String(input)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll("'", "&#x27;")
    .replaceAll("\"", "&quot;");
}

/**
 * @param {TemplateStringsArray | string} str
 * @param {...unknown} values
 * @returns {DocumentFragment}
 */
export function html(str, ...values) {
  const tmpl = document.createElement("template");
  if (typeof str === "object" && "raw" in str) {
    str = String.raw(str, ...values.map(htmlescape));
  }
  tmpl.innerHTML = str;
  return tmpl.content;
}

/**
 * @param {TemplateStringsArray | string} str
 * @param {...unknown} values
 * @returns {CSSStyleSheet}
 */
export function css(str, ...values) {
  const stylesheet = new CSSStyleSheet();
  if (typeof str === "object" && "raw" in str) {
    str = String.raw(str, ...values);
  }
  stylesheet.replaceSync(str);
  return stylesheet;
}

/**
 * @param {TemplateStringsArray | string} str
 * @param {...unknown} values
 * @returns {CSSStyleDeclaration}
 */
export function style(str, ...values) {
  const styleDeclaration = new CSSStyleDeclaration();
  if (typeof str === "object" && "raw" in str) {
    str = String.raw(str, ...values);
  }
  styleDeclaration.cssText = str;
  return styleDeclaration;
}

/**
 * @param {unknown} el
 * @returns {HTMLElement | null}
 */
export function asHtml(el) {
  return el instanceof HTMLElement ? el : null;
}

/**
 * @param {ParentNode} root
 * @param {string} selector
 * @param {Element | null} current
 * @param {object} options
 * @param {boolean} [options.wrap]
 */
export function next(root, selector, current, options = {}) {
  return traverse("next", root, selector, current, options);
}

/**
 * @param {ParentNode} root
 * @param {string} selector
 * @param {Element | null} current
 * @param {object} options
 * @param {boolean} [options.wrap]
 */
export function prev(root, selector, current, options = {}) {
  return traverse("previous", root, selector, current, options);
}

/**
 * @callback KeyboardEventListener
 * @param {KeyboardEvent} e
 * @returns {void}
 */

/**
 * @param {Record<string, KeyboardEventListener>} hotkeys
 * @returns {KeyboardEventListener}
 */
export function hotkey(hotkeys) {
  const alt = 0b1;
  const ctrl = 0b10;
  const meta = 0b100;
  const shift = 0b1000;
  /** @type {Record<string, Record<number, KeyboardEventListener>>} */
  const handlers = {};
  const modifiersOf = (event) =>
    ~~(event.altKey && alt) |
    ~~(event.ctrlKey && ctrl) |
    ~~(event.metaKey && meta) |
    ~~(event.shiftKey && shift);
  const parse = (hotkeySpec) => {
    const tokens = hotkeySpec.split("+");
    const key = /** @type {string} */ (tokens.pop());
    let modifiers = 0 | 0;
    for (const token of tokens) {
      switch (token.toLowerCase()) {
        case "alt":
          modifiers |= alt;
          break;
        case "ctrl":
          modifiers |= ctrl;
          break;
        case "meta":
          modifiers |= meta;
          break;
        case "shift":
          modifiers |= shift;
          break;
      }
    }
    return [key, modifiers];
  };

  for (const [hotkeySpec, handler] of Object.entries(hotkeys)) {
    const [key, modifiers] = parse(hotkeySpec);
    (handlers[key] ??= new Array(8))[modifiers] = handler;
  }

  return (event) => handlers[event.key]?.[modifiersOf(event)]?.(event);
}

/**
 * @template {unknown[]} TArgs
 * @param {number} timeoutMS
 * @param {(...args: TArgs) => void} f
 * @param {object} [options]
 * @param {"leading" | "trailing"} [options.mode]
 * @returns {(...args: TArgs) => void}
 */
export function debounce(timeoutMS, f, { mode = "trailing" } = {}) {
  /** @type {number | null} */
  let timeout = null;
  return (...args) => {
    if (timeout) clearTimeout(timeout);
    else if (mode === "leading") f(...args);

    timeout = setTimeout(() => {
      if (mode === "trailing") f(...args);
      timeout = null;
    }, timeoutMS);
  };
}

/**
 * @template TOptions
 * @param {string} selector
 * @param {BehaviorInit<TOptions>} init
 * @returns {Behavior<TOptions>}
 */
export function behavior(selector, init) {
  const initialized = new WeakSet();
  return (subtree = document, options = {}) => {
    const root = /** @type {Root} */ (subtree.getRootNode());
    $$(subtree, selector).forEach((el) => {
      if (initialized.has(el)) return;
      initialized.add(el);
      init(el, { options, root });
    });
  };
}

/**
 * @template TData
 * @typedef {object} Repeater
 * @property {(datum: TData) => string} idOf
 * @property {(datum: TData, ctx: { id: string }) => ChildNode} create
 * @property {(el: Element, datum: TData) => Element | null} update
 */

/**
 * @template TData
 * @param {ParentNode} container
 * @param {Repeater<TData>} rep
 */
export function repeater(container, rep) {
  return (dataset) => {
    /** @type {ChildNode | null} */
    let cursor = null;

    const append = (...nodes) => {
      const oldCursor = cursor;
      cursor = /** @type {ChildNode} */ (nodes.at(-1));
      if (cursor instanceof DocumentFragment) cursor = cursor.lastChild;
      if (oldCursor) oldCursor.after(...nodes);
      else container.prepend(...nodes);
    };

    const clearAfter = (currentCursor) => {
      if (currentCursor) {
        while (currentCursor.nextSibling) currentCursor.nextSibling.remove();
      } else {
        container.replaceChildren();
      }
    };

    const root = /** @type {Root} */ (container.getRootNode());

    for (const datum of dataset) {
      const id = rep.idOf(datum);
      const existing = root.getElementById(id);
      if (existing) append(rep.update?.(existing, datum) ?? existing);
      else append(rep.create(datum, { id }));
    }

    clearAfter(cursor);
  };
}
// @license-end
