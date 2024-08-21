// ==UserScript==
// @name         Cosmetic Ad Block for Bromite{{if .isLite}} (Lite){{end}}
// @namespace    xarantolus
// @version      {{.version}}
// @description  Blocks annoying elements in {{if .isLite}}top {{.topDomainCount}} domains{{else}}pages{{end}}, sourced from many different filter lists
// @author       xarantolus
// @match        *://*/*
// @grant        none
// @run-at       document-start
// @homepage     https://userscripts.010.one
// @url_source   https://github.com/xarantolus/bromite-userscripts/releases/latest/download/cosmetic{{if .isLite}}-lite{{end}}.user.js
// ==/UserScript==
/// @stats {{.statistics}}
{
    let log = function (...data) {
        console.log("[Cosmetic filters by xarantolus (v{{.version}} {{if .isLite}}lite{{else}}full{{end}})]:", ...data);
    }


    function injectStyle(cssStyle) {
        let style = document.createElement('style');
        style.type = 'text/css';
        style.innerHTML = cssStyle;
        document.getElementsByTagName('head')[0].appendChild(style);
    }

    let deduplicatedStrings = {{.deduplicatedStrings }};
    let injectionRules = {{.injectionRules }};
    let injectionExceptions = {{.injectionExceptions }};
    let rules = {{.rules }};
    let exceptions = {{.exceptions }};
    let defaultRules = rules[""];
    let defaultExceptions = exceptions[""];
    let defaultInjections = injectionRules[""];
    let defaultInjectionExceptions = injectionExceptions[""];

    function findRules(rules, host) {
        let domainSplit = host.split(".");

        let output = [];

        for (i in rules) {
            let ruleSplit = i.split(",");

            let allTilded = true;
            for (let j = 0; j < ruleSplit.length; j++) {
                if (!ruleSplit[j].startsWith("~")) {
                    allTilded = false;
                    break;
                }
            }

            if (allTilded) {
                log("Checking if we got an all-tilded rule");

                let foundTilded = false;

                for (let k = 0; k < domainSplit.length - 1; k++) {
                    let tilded = "~" + domainSplit.slice(k, domainSplit.length).join(".").toLowerCase();

                    if (ruleSplit.includes(tilded)) {
                        foundTilded = true;
                        break;
                    }
                }

                if (!foundTilded) {
                    output.push(i);
                }
            } else {
                for (let k = 0; k < domainSplit.length - 1; k++) {
                    let domain = domainSplit.slice(k, domainSplit.length).join(".").toLowerCase();

                    if (ruleSplit.includes(domain)) {
                        log("Checking if we got a rule for", domain);

                        let foundTilded = false;

                        for (let l = 0; l < domainSplit.length - 1; l++) {
                            let tilded = "~" + domainSplit.slice(l, domainSplit.length).join(".").toLowerCase();

                            if (ruleSplit.includes(tilded)) {
                                foundTilded = true;
                                break;
                            }
                        }

                        if (!foundTilded) {
                            output.push(i);
                        }
                    }
                }
            }
        }

        return output;
    }


    function getRules(host) {
        let output = [];

        let ruleKeys = findRules(rules, host);

        for (let i = 0; i < ruleKeys.length; i++) {
            let rule = rules[ruleKeys[i]];
            if (rule != null) {
                if (typeof rule === 'number') {
                    // the selector is saved at this index in the deduplicatedRules array
                    let realRule = deduplicatedStrings[rule];
                    log("Found deduplicated rule", rule, "for domain", ruleKeys[i]);
                    output.push({ "s": realRule });
                } else {
                    // It's a string that directly defines the selector
                    log("Found normal rule for domain", ruleKeys[i]);
                    output.push({ "s": rule });
                }
            }
        }

        let exceptionKeys = findRules(exceptions, host)

        for (let i = 0; i < exceptionKeys.length; i++) {
            let exception = exceptions[exceptionKeys[i]];
            if (exception != null) {
                if (typeof exception === 'number') {
                    // the exception is saved at this index in the deduplicatedExceptions array
                    let realException = deduplicatedStrings[exception];
                    log("Found deduplicated exception", exception, "for domain", exceptionKeys[i]);
                    output.push({ "e": realException });
                } else {
                    // It's a string that directly defines the selector
                    log("Found normal exception for domain", exceptionKeys[i]);
                    output.push({ "e": exception });
                }
            }
        }

        let injectionKeys = findRules(injectionRules, host)

        for (let i = 0; i < injectionKeys.length; i++) {
            let injection = injectionRules[injectionKeys[i]];
            if (injection != null) {
                if (typeof injection === 'number') {
                    let realInjection = deduplicatedStrings[injection];
                    log("Found deduplicated injection", injection, "for domain", injectionKeys[i]);
                    output.push({ "i": realInjection })
                } else {
                    log("Found normal injection for domain", injectionKeys[i]);
                    output.push({ "i": injection });
                }
            }
        }

        let injectionExceptionKeys = findRules(injectionExceptions, host)

        for (let i = 0; i < injectionExceptionKeys.length; i++) {
            let injectionException = injectionExceptionKeys[injectionExceptionKeys[i]];
            if (injectionException != null) {
                if (typeof injectionException === 'number') {
                    let realInjectionException = deduplicatedStrings[injectionException];
                    log("Found deduplicated injection exception", injectionException, "for domain", injectionExceptionKeys[i]);
                    output.push({ "x": realInjectionException })
                } else {
                    log("Found normal injection exception for domain", injectionExceptionKeys[i]);
                    output.push({ "x": injectionException });
                }
            }
        }

        if (defaultRules) { output.push({ "s": defaultRules, isDefault: true }) };
        if (defaultExceptions) { output.push({ "e": defaultExceptions, isDefault: true }) };
        if (defaultInjections) { output.push({ "i": defaultInjections, isDefault: true }) };
        if (defaultInjectionExceptions) { output.push({ "x": defaultInjectionExceptions, isDefault: true }) };

        return output;
    }

    let hiddenStyle = "display:none!important;min-height:0!important;height:0!important;z-index:-99999!important;visibility:hidden!important;width:0!important;min-width:0!important;overflow:hidden!important";
    let hideRules = "{" + hiddenStyle + "}"

    let foundRules = getRules(location.host);

    log("Found", foundRules.length, "rules to inject");

    let notSelector = ":not(" + foundRules.filter(r => r["e"] != null)
        .map(r => r["e"]).join(",") + ")"

    // unlikely but possible
    if (notSelector === ":not()") {
        notSelector = "";
    }

    let hiddenElementsSelector = ":is(" + foundRules.filter(r => r["s"] != null)
        .map(r => r["s"]).join(",") + ")" + notSelector + hideRules;

    let cssInjections = foundRules.filter(r => r["i"] != null).map(r => r["i"]).join("");
    let cssInjectionExceptions = foundRules.filter(r => r["x"] != null).map(r => r["x"]).join("|");

    log("found injection exception rules:", cssInjectionExceptions);

    if (cssInjectionExceptions !== "") {
        let cssInjectionExceptionsRegex = new RegExp(cssInjectionExceptions, "g");

        cssInjections.replace(cssInjectionExceptionsRegex, "");
    }

    log("injection string after exception:", cssInjections);

    // let pageSpecificNotSelector = ":not(" + foundRules.filter(r => r["e"] != null && !r.isDefault)
    //     .map(r => r["e"]).join(",") + ")"

    let pageSpecificSelectors = ":is(" + foundRules.filter(r => r["s"] != null && !r.isDefault)
        .map(r => r["s"]).join(",") + ")" + notSelector;

    log("Page specific selectors:", (pageSpecificSelectors || "(none)"))

    // Source: https://stackoverflow.com/a/61747276
    function elementReady(selector) {
        return new Promise((resolve, reject) => {
            const el = document.querySelector(selector);
            if (el) { resolve(el); }
            new MutationObserver((mutationRecords, observer) => {
                // Query for elements matching the specified selector
                Array.from(document.querySelectorAll(selector)).forEach((element) => {
                    resolve(element);
                    //Once we have resolved we don't need the observer anymore.
                    observer.disconnect();
                });
            })
                .observe(document.documentElement, {
                    childList: true,
                    subtree: false // This was changed to "false" since we only need "head", a direct descendant of the document element
                });
        });
    }

    function hidePageSpecificElements(reason) {
        if (pageSpecificSelectors.startsWith(":is()")) return;

        log("Searching for elements (" + reason + ")")
        let elems = [...document.querySelectorAll(pageSpecificSelectors)];
        elems.forEach(function (elem) {
            elem.setAttribute("style", hiddenStyle);
        });
        log("Tried hiding", elems.length, "page-specific elements");
    }

    // Now we have hidden a lot of stuff using rules. However, some sites still display elements
    // because they look like <span class="ad" style="display:block">
    // This means that the !important from our css declaration above will not work on these elements (as direct styles take precedence)
    // We need to replace the style of all elements with this selector
    // When the HTML has finished parsing:
    window.addEventListener('DOMContentLoaded', function () {
        hidePageSpecificElements("DOMContentLoaded");

        setTimeout(() => hidePageSpecificElements("DOMContentLoaded + 1000ms"), 1000);
    });
    // And after the page is fully loaded, we do a bunch of checks within the first second or so.
    // If a page pops up a cookie popup after the page has loaded, this one will also defeat it
    window.addEventListener('load', function () {
        hidePageSpecificElements("load - initial");

        function to(offset) {
            let ms = offset * 500;
            setTimeout(() => hidePageSpecificElements("load + " + ms + "ms"), ms);
        };
        for (let i = 1; i <= 5; i++) {
            to(i);
        }
    })

    elementReady('head').then((_) => {
        injectStyle(hiddenElementsSelector);
        log("Injected combined style");

        if (cssInjections.length > 0) {
            injectStyle(cssInjections);
            log("Also injected additional styles (usually fixes for scrolling issues)")
        }
    });
}
