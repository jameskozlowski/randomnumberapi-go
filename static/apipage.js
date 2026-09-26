document.querySelectorAll('[data-api-path]').forEach((link) => {
    const path = link.dataset.apiPath;
    link.href = path;
    link.textContent = window.location.origin + path;
});
