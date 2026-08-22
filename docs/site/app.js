(function () {
  'use strict';

  const select = document.getElementById('case-select');
  const view = document.getElementById('evidence-view');
  const status = document.getElementById('copy-status');
  let cases = [];

  function textAll(selector, value) {
    document.querySelectorAll(selector).forEach(function (element) {
      element.textContent = value;
    });
  }

  function configureProject(project) {
    document.title = project.name + ' Evidence Playground';
    document.documentElement.style.setProperty('--accent', project.accent);
    document.documentElement.style.setProperty('--accent-soft', project.accentSoft);
    textAll('[data-project-name]', project.name);
    textAll('[data-project-title]', project.title);
    textAll('[data-project-description]', project.description);
    textAll('[data-project-schema]', project.schema);
    textAll('[data-project-version]', project.version);
    textAll('[data-project-boundary]', project.boundary);
    textAll('[data-project-local]', project.local);

    document.querySelector('[data-repo-link]').href = project.repository;
    document.querySelector('[data-release-link]').href = project.release;
    const active = document.querySelector('[data-family="' + project.slug + '"]');
    if (active) active.setAttribute('aria-current', 'page');
  }

  function renderCase(item) {
    document.getElementById('case-kicker').textContent = item.kicker;
    document.getElementById('case-title').textContent = item.title;
    document.getElementById('case-status').textContent = item.status;
    document.getElementById('case-summary').textContent = item.summary;
    document.getElementById('case-source').textContent = item.source;
    document.getElementById('case-command').textContent = item.command;
    document.getElementById('case-json').textContent = JSON.stringify(item.evidence, null, 2);

    const metrics = document.getElementById('case-metrics');
    metrics.replaceChildren();
    item.metrics.forEach(function (metric) {
      const wrapper = document.createElement('div');
      const term = document.createElement('dt');
      const value = document.createElement('dd');
      term.textContent = metric.label;
      value.textContent = metric.value;
      wrapper.append(term, value);
      metrics.append(wrapper);
    });

    const highlights = document.getElementById('case-highlights');
    highlights.replaceChildren();
    item.highlights.forEach(function (highlight) {
      const row = document.createElement('li');
      row.textContent = highlight;
      highlights.append(row);
    });
  }

  async function copy(value, message) {
    try {
      await navigator.clipboard.writeText(value);
      status.textContent = message;
    } catch (error) {
      status.textContent = 'Copy was blocked. Select the text manually.';
    }
  }

  select.addEventListener('change', function () {
    renderCase(cases[Number(select.value)]);
    status.textContent = '';
  });
  document.getElementById('copy-command').addEventListener('click', function () {
    copy(document.getElementById('case-command').textContent, 'CLI command copied.');
  });
  document.getElementById('copy-json').addEventListener('click', function () {
    copy(document.getElementById('case-json').textContent, 'Evidence JSON copied.');
  });

  fetch('cases.json', { cache: 'no-store' })
    .then(function (response) {
      if (!response.ok) throw new Error('Unable to load committed evidence.');
      return response.json();
    })
    .then(function (payload) {
      configureProject(payload.project);
      cases = payload.cases;
      cases.forEach(function (item, index) {
        const option = document.createElement('option');
        option.value = String(index);
        option.textContent = item.title;
        select.append(option);
      });
      renderCase(cases[0]);
      view.setAttribute('aria-busy', 'false');
    })
    .catch(function (error) {
      view.setAttribute('aria-busy', 'false');
      document.getElementById('case-title').textContent = 'Evidence unavailable';
      document.getElementById('case-summary').textContent = error.message;
    });
})();
