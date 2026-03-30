// @license magnet:?xt=urn:btih:0b31508aeb0634b347b8270c7bee4d411b5d4109&dn=agpl-3.0.txt AGPL-3.0-or-later
const importFileInput = document.getElementById("import-file");
const importStatus = document.getElementById("import-file-status");
const importForm = document.querySelector('form[action="/app-settings/import"]');

if (importFileInput && importStatus) {
  importFileInput.addEventListener("change", (event) => {
    const target = event.currentTarget;
    const file = target.files && target.files[0];
    if (!file) {
      importStatus.textContent = "Choose a JSON backup file to import it directly.";
      return;
    }

    importStatus.textContent = `Selected ${file.name} (${Math.max(1, Math.round(file.size / 1024))} KB).`;
  });
}

if (importForm) {
  importForm.addEventListener("submit", (event) => {
    const ok = window.confirm("Importing a snapshot replaces the current supported data and signs out every active session. Continue?");
    if (!ok) {
      event.preventDefault();
    }
  });
}

// @license-end
