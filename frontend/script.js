function fetchTree() {
  const url = document.getElementById('repo-url').value;
  const output = document.getElementById('output');
  output.textContent = 'Loading...';

  fetch(`/tree?url=${encodeURIComponent(url)}`)
    .then(res => {
      if (!res.ok) throw new Error("Gagal mengambil data");
      return res.json();
    })
    .then(data => {
      output.textContent = renderTree(data, 0);
    })
    .catch(err => {
      output.textContent = 'Error: ' + err.message;
    });
}

function renderTree(node, depth) {
  let result = '  '.repeat(depth) + (node.is_dir ? '📁 ' : '📄 ') + node.name + '\n';
  if (node.children) {
    for (let child of node.children) {
      result += renderTree(child, depth + 1);
    }
  }
  return result;
}
