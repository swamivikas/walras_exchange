let canvas, ctx;

const nodes = [
  { id: 'gateway', label: 'Walras Gateway (Go)', x: 0.5, y: 0.45, radius: 12, color: '#38bdf8', pulse: 0 },
  { id: 'buyer', label: 'Enterprise Buyer', x: 0.2, y: 0.65, radius: 8, color: '#10b981', pulse: 0 },
  { id: 'seller1', label: 'GPU-01 (RTX 4090)', x: 0.8, y: 0.35, radius: 9, color: '#f59e0b', pulse: 0 },
];

let packets = [];

export function initRadar() {
  canvas = document.getElementById('topologyCanvas');
  if (!canvas) return;
  ctx = canvas.getContext('2d');

  function resize() {
    canvas.width = canvas.parentElement.clientWidth;
    canvas.height = canvas.parentElement.clientHeight;
  }
  window.addEventListener('resize', resize);
  resize();

  requestAnimationFrame(drawTopology);
}

export function spawnPacket(fromId, toId, color = '#00f0ff') {
  if (!canvas) return;
  const fromNode = nodes.find(n => n.id === fromId);
  const toNode = nodes.find(n => n.id === toId);
  if (fromNode && toNode) {
    packets.push({
      fromX: fromNode.x * canvas.width,
      fromY: fromNode.y * canvas.height,
      toX: toNode.x * canvas.width,
      toY: toNode.y * canvas.height,
      progress: 0,
      speed: 0.04,
      color: color
    });
  }
}

function drawTopology() {
  if (!ctx || !canvas) return;
  ctx.clearRect(0, 0, canvas.width, canvas.height);

  // Connection Grid
  ctx.strokeStyle = 'rgba(0, 240, 255, 0.15)';
  ctx.setLineDash([4, 4]);
  const gw = nodes[0];
  const gwX = gw.x * canvas.width;
  const gwY = gw.y * canvas.height;

  for (let i = 1; i < nodes.length; i++) {
    const n = nodes[i];
    const nX = n.x * canvas.width;
    const nY = n.y * canvas.height;
    ctx.beginPath();
    ctx.moveTo(gwX, gwY);
    ctx.lineTo(nX, nY);
    ctx.stroke();
  }
  ctx.setLineDash([]);

  // Moving Packets
  for (let i = packets.length - 1; i >= 0; i--) {
    const p = packets[i];
    p.progress += p.speed;
    const curX = p.fromX + (p.toX - p.fromX) * p.progress;
    const curY = p.fromY + (p.toY - p.fromY) * p.progress;

    ctx.fillStyle = p.color;
    ctx.shadowColor = p.color;
    ctx.shadowBlur = 10;
    ctx.beginPath();
    ctx.arc(curX, curY, 3.5, 0, Math.PI * 2);
    ctx.fill();
    ctx.shadowBlur = 0;

    if (p.progress >= 1) packets.splice(i, 1);
  }

  // Draw Nodes
  nodes.forEach(n => {
    const x = n.x * canvas.width;
    const y = n.y * canvas.height;
    n.pulse = (n.pulse + 0.03) % (Math.PI * 2);
    const pulseRadius = n.radius + Math.sin(n.pulse) * 4;

    ctx.strokeStyle = n.color;
    ctx.lineWidth = 1;
    ctx.beginPath();
    ctx.arc(x, y, pulseRadius + 3, 0, Math.PI * 2);
    ctx.stroke();

    ctx.fillStyle = n.color;
    ctx.shadowColor = n.color;
    ctx.shadowBlur = 15;
    ctx.beginPath();
    ctx.arc(x, y, n.radius, 0, Math.PI * 2);
    ctx.fill();
    ctx.shadowBlur = 0;

    ctx.fillStyle = '#f8fafc';
    ctx.font = '500 11px "Plus Jakarta Sans", sans-serif';
    ctx.textAlign = 'center';
    ctx.fillText(n.label, x, y + n.radius + 14);
  });

  requestAnimationFrame(drawTopology);
}
