const containerList = document.getElementById('container-list');

async function loadContainers() {
  try {
    const result = await window.otter.run('otter', ['reg', 'ls', '--json']);
    const entries = JSON.parse(result.stdout);

    if (entries.length === 0) {
      containerList.innerHTML = '<p class="empty-state">No containers found</p>';
      return;
    }

    containerList.innerHTML = entries
      .map(
        (e) => `
      <div class="container-entry">
        <span class="name">${e.name}</span>
        <span class="status">${e.pulled ? 'Pulled' : 'Not pulled'}</span>
      </div>
    `
      )
      .join('');
  } catch (err) {
    containerList.innerHTML = `<p class="empty-state">Error: ${err.message}</p>`;
  }
}

loadContainers();
