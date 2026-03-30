// @license magnet:?xt=urn:btih:87f119ba0b429ba17a44b4bffcab33165ebdacc0&dn=freebsd.txt BSD-2-Clause
/// a tabs library.

//@deno-types=./19.ts
import {
  $,
  $$,
  on,
  attr,
  next,
  prev,
  asHtml,
  hotkey,
  behavior,
  identify,
  dispatch,
} from "./missing-19.js";

/**
 * @param {Element} tablist
 * @returns {HTMLElement[]}
 */
const tabsOf = (tablist) => $$(tablist, "[role=tab]");

/**
 * @param {Element} tablist
 * @returns {HTMLElement | null}
 */
const currentTab = (tablist) => $(tablist, "[role=tab][aria-selected=true]");

/**
 * @param {Element} tab
 * @param {Document | ShadowRoot} root
 * @returns {HTMLElement | null}
 */
const tabPanelOf = (tab, root) => {
  const id = attr(tab, "aria-controls");
  if (id === null) {
    console.error("Tab", tab, "has no associated tabpanel");
    return null;
  }
  return root.getElementById(id);
};

/**
 * @param {Document | ShadowRoot} root
 * @param {Element} tablist
 * @param {HTMLElement | null} tab
 * @param {{ focusTab?: boolean }} [options]
 */
const switchTab = (root, tablist, tab, { focusTab = true } = {}) => {
  if (!tab) return;
  const curtab = currentTab(tablist);

  if (curtab) {
    attr(curtab, { ariaSelected: false, tabindex: -1 });
    const currentPanel = tabPanelOf(curtab, root);
    if (currentPanel) currentPanel.hidden = true;
  }

  attr(tab, { ariaSelected: true, tabindex: 0 });
  const tabpanel = tabPanelOf(tab, root);
  if (tabpanel) tabpanel.hidden = false;

  if (focusTab) tab.focus();
  tablist.tabIndex = -1;

  dispatch(curtab, "missing-switch-away", { to: tab });
  dispatch(tab, "missing-switch-to", { from: curtab });
  dispatch(tablist, "missing-change", { from: curtab, to: tab });
};

export const tablist = behavior("[role=tablist]", (tablist, { root }) => {
  if (!(tablist instanceof HTMLElement)) return;

  tablist.tabIndex = 0;
  tabsOf(tablist).forEach((tab) => {
    tab.tabIndex = -1;
    const panel = tabPanelOf(tab, root);
    if (!panel) return;
    panel.setAttribute("aria-labelledby", identify(tab));
  });

  if (!(tablist.hasAttribute("aria-labelledby") || tablist.hasAttribute("aria-label"))) {
    console.error("Tab list", tablist, "has no accessible name (aria-label or aria-labelledby)");
  }

  switchTab(root, tablist, currentTab(tablist), { focusTab: false });

  on(tablist, "focus", () => currentTab(tablist)?.focus());
  on(tablist, "click", (event) =>
    switchTab(root, tablist, asHtml(asHtml(event.target)?.closest("[role=tab]"))),
  );
  on(tablist, "focusin", (event) =>
    switchTab(root, tablist, asHtml(asHtml(event.target)?.closest("[role=tab]"))),
  );

  on(
    tablist,
    "keydown",
    hotkey({
      ArrowRight: (event) => asHtml(next(tablist, "[role=tab]", asHtml(event.target)))?.focus(),
      ArrowLeft: (event) => asHtml(prev(tablist, "[role=tab]", asHtml(event.target)))?.focus(),
      Home: () => tabsOf(tablist).at(0)?.focus(),
      End: () => tabsOf(tablist).at(-1)?.focus(),
    }),
  );
});

tablist(document);
if (typeof document !== "undefined") {
  document.addEventListener("htmx:afterSwap", (event) => {
    const target = event.target instanceof Element ? event.target : document;
    tablist(target);
  });
}
export default tablist;
// @license-end
