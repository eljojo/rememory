/**
 * ReMemory Data Flow Animation
 *
 * Educational visualization showing how Shamir's Secret Sharing works:
 * - A file is encrypted and split into 5 shares
 * - Shares are distributed to 5 friends
 * - Any 3 of 5 friends can recover the original
 * - Each loop shows a different combination of 3 friends
 */

(function() {
  'use strict';

  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // Friend names (matching the sample names from maker)
  const FRIEND_NAMES = ['Catalina', 'Matthias', 'Sophie', 'Joaquín', 'Emma'];

  // Colors
  const COLORS = {
    file: '#667eea',
    fileLight: '#8b9df0',
    encrypt: '#764ba2',
    split: '#9b59b6',
    friend: ['#e74c3c', '#3498db', '#2ecc71', '#f39c12', '#9b59b6'],
    friendInactive: '#bdc3c7',
    gold: '#ffd700',
    goldLight: '#ffec8b',
    line: '#95a5a6',
    lineActive: '#667eea',
    text: '#2c3e50',
    textLight: '#7f8c8d',
    recovered: '#27ae60'
  };

  // Easing functions
  const ease = {
    inOut: t => t < 0.5 ? 2 * t * t : 1 - Math.pow(-2 * t + 2, 2) / 2,
    out: t => 1 - Math.pow(1 - t, 3),
    in: t => t * t * t
  };

  class DataFlowCanvas {
    constructor(canvas) {
      this.canvas = canvas;
      this.ctx = canvas.getContext('2d');
      this.running = false;
      this.time = 0;
      this.phase = 'split'; // 'split', 'wait', 'recover', 'complete'
      this.phaseTime = 0;
      this.loopCount = 0;

      // Which 3 friends are active this recovery cycle
      this.activeFriends = [0, 1, 2];

      // All possible 3-of-5 combinations (10 total)
      this.combinations = this.generateCombinations(5, 3);
      this.combinationIndex = 0;

      // Animation state
      this.particles = [];
      this.splitProgress = 0;
      this.sharePositions = [0, 0, 0, 0, 0]; // 0-1 progress to friends
      this.recoverPositions = [0, 0, 0]; // 0-1 progress back to center
      this.combineProgress = 0;
      this.fileRecovered = false;
      this.hasDistributed = false; // Track if initial distribution is done

      // Layout will be set on resize
      this.layout = {};

      this.resize();
      window.addEventListener('resize', () => this.resize());
    }

    generateCombinations(n, k) {
      const result = [];
      const combo = [];

      function generate(start) {
        if (combo.length === k) {
          result.push([...combo]);
          return;
        }
        for (let i = start; i < n; i++) {
          combo.push(i);
          generate(i + 1);
          combo.pop();
        }
      }

      generate(0);
      return result;
    }

    resize() {
      const rect = this.canvas.getBoundingClientRect();
      const dpr = window.devicePixelRatio || 1;
      this.canvas.width = rect.width * dpr;
      this.canvas.height = rect.height * dpr;
      this.ctx.scale(dpr, dpr);
      this.width = rect.width;
      this.height = rect.height;

      this.calculateLayout();
    }

    calculateLayout() {
      const w = this.width;
      const h = this.height;
      const padding = 40;
      const usableWidth = w - padding * 2;
      const usableHeight = h - padding * 2;
      const centerY = h / 2;

      // Horizontal positions (left to right flow)
      const fileX = padding + usableWidth * 0.06;
      const encryptX = padding + usableWidth * 0.18;
      const splitX = padding + usableWidth * 0.32;
      const friendsX = padding + usableWidth * 0.48;
      const combineX = padding + usableWidth * 0.72;
      const recoveredX = padding + usableWidth * 0.90;

      // Friend vertical positions (spread out)
      const friendSpacing = Math.min(usableHeight / 6, 50);
      const friendsStartY = centerY - friendSpacing * 2;

      this.layout = {
        // Split phase (top flow)
        file: { x: fileX, y: centerY - 40 },
        encrypt: { x: encryptX, y: centerY - 40 },
        split: { x: splitX, y: centerY - 40 },

        // Friends (right side, vertical stack)
        friends: Array.from({ length: 5 }, (_, i) => ({
          x: friendsX,
          y: friendsStartY + i * friendSpacing
        })),

        // Recovery phase (bottom flow)
        combine: { x: combineX, y: centerY + 50 },
        recovered: { x: recoveredX, y: centerY + 50 },

        // Node sizes
        nodeRadius: Math.min(22, usableWidth * 0.035),
        smallRadius: Math.min(16, usableWidth * 0.025),

        // For labels
        labelOffset: 28
      };
    }

    nextCombination() {
      this.combinationIndex = (this.combinationIndex + 1) % this.combinations.length;
      this.activeFriends = this.combinations[this.combinationIndex];
    }

    resetAnimation() {
      this.phaseTime = 0;
      this.recoverPositions = [0, 0, 0];
      this.combineProgress = 0;
      this.fileRecovered = false;
      this.particles = [];
      this.nextCombination();
      this.loopCount++;

      if (this.hasDistributed) {
        // After first distribution, only loop the recovery phase
        this.phase = 'recover';
      } else {
        // First time: start from beginning
        this.phase = 'split';
        this.splitProgress = 0;
        this.sharePositions = [0, 0, 0, 0, 0];
      }
    }

    update(dt) {
      if (prefersReducedMotion) {
        // Show final state
        this.phase = 'complete';
        this.splitProgress = 1;
        this.sharePositions = [1, 1, 1, 1, 1];
        this.recoverPositions = [1, 1, 1];
        this.combineProgress = 1;
        this.fileRecovered = true;
        return;
      }

      this.time += dt;
      this.phaseTime += dt;

      const SPLIT_DURATION = 3000;
      const DISTRIBUTE_DURATION = 2000;
      const WAIT_DURATION = this.hasDistributed ? 800 : 1500; // Shorter wait on loops
      const RECOVER_DURATION = 2500;
      const COMBINE_DURATION = 1500;
      const COMPLETE_DURATION = 2500;

      switch (this.phase) {
        case 'split':
          // Animate file → encrypt → split
          this.splitProgress = Math.min(1, this.phaseTime / SPLIT_DURATION);

          if (this.splitProgress >= 1) {
            this.phase = 'distribute';
            this.phaseTime = 0;
          }
          break;

        case 'distribute':
          // Animate shares flowing to friends
          const distProgress = this.phaseTime / DISTRIBUTE_DURATION;
          for (let i = 0; i < 5; i++) {
            // Stagger the distribution (smaller delays so all complete)
            const delay = i * 0.05;
            const p = Math.max(0, Math.min(1, (distProgress - delay) / (0.8 - delay)));
            this.sharePositions[i] = ease.out(p);
          }

          if (distProgress >= 1) {
            // Ensure all positions are fully 1
            for (let i = 0; i < 5; i++) {
              this.sharePositions[i] = 1;
            }
            this.hasDistributed = true; // Mark that distribution is complete
            this.phase = 'wait';
            this.phaseTime = 0;
          }
          break;

        case 'wait':
          // Brief pause with "time passes" indication
          if (this.phaseTime >= WAIT_DURATION) {
            this.phase = 'recover';
            this.phaseTime = 0;
          }
          break;

        case 'recover':
          // Animate active friends' shares coming back
          const recProgress = this.phaseTime / RECOVER_DURATION;
          for (let i = 0; i < 3; i++) {
            const delay = i * 0.15;
            const p = Math.max(0, Math.min(1, (recProgress - delay) / 0.6));
            this.recoverPositions[i] = ease.out(p);
          }

          if (recProgress >= 1) {
            this.phase = 'combine';
            this.phaseTime = 0;
          }
          break;

        case 'combine':
          // Animate combination with golden seams
          this.combineProgress = Math.min(1, this.phaseTime / COMBINE_DURATION);

          if (this.combineProgress >= 0.7 && !this.fileRecovered) {
            this.fileRecovered = true;
          }

          if (this.combineProgress >= 1) {
            this.phase = 'complete';
            this.phaseTime = 0;
          }
          break;

        case 'complete':
          // Show completed state, then fade out and reset
          // Fade out the recovery elements in the second half
          if (this.phaseTime > COMPLETE_DURATION * 0.5) {
            const fadeProgress = (this.phaseTime - COMPLETE_DURATION * 0.5) / (COMPLETE_DURATION * 0.5);
            this.combineProgress = 1 - fadeProgress;
            for (let i = 0; i < 3; i++) {
              this.recoverPositions[i] = 1 - fadeProgress;
            }
          }

          if (this.phaseTime >= COMPLETE_DURATION) {
            this.resetAnimation();
          }
          break;
      }

      // Update particles
      this.updateParticles(dt);
    }

    updateParticles(dt) {
      // Add new particles along active paths
      if (Math.random() < 0.3 && this.phase !== 'wait' && this.phase !== 'complete') {
        this.addParticle();
      }

      // Update existing particles
      for (let i = this.particles.length - 1; i >= 0; i--) {
        const p = this.particles[i];
        p.life -= dt * 0.001;
        p.progress += p.speed * dt * 0.001;

        if (p.life <= 0 || p.progress > 1) {
          this.particles.splice(i, 1);
        }
      }
    }

    addParticle() {
      const { layout } = this;
      let start, end, color;

      if (this.phase === 'split' && this.splitProgress < 0.3) {
        start = layout.file;
        end = layout.encrypt;
        color = COLORS.file;
      } else if (this.phase === 'split' && this.splitProgress < 0.6) {
        start = layout.encrypt;
        end = layout.split;
        color = COLORS.encrypt;
      } else if (this.phase === 'distribute') {
        const friendIdx = Math.floor(Math.random() * 5);
        start = layout.split;
        end = layout.friends[friendIdx];
        color = COLORS.friend[friendIdx];
      } else if (this.phase === 'recover') {
        const activeIdx = Math.floor(Math.random() * 3);
        const friendIdx = this.activeFriends[activeIdx];
        start = layout.friends[friendIdx];
        end = layout.combine;
        color = COLORS.friend[friendIdx];
      } else if (this.phase === 'combine') {
        start = layout.combine;
        end = layout.recovered;
        color = COLORS.gold;
      } else {
        return;
      }

      this.particles.push({
        startX: start.x,
        startY: start.y,
        endX: end.x,
        endY: end.y,
        progress: 0,
        speed: 0.3 + Math.random() * 0.2,
        life: 1,
        color,
        size: 3 + Math.random() * 2
      });
    }

    draw() {
      const { ctx, layout } = this;
      ctx.clearRect(0, 0, this.width, this.height);

      // Draw connection lines first (behind everything)
      this.drawConnections();

      // Draw particles
      this.drawParticles();

      // Draw nodes
      this.drawSplitPhase();
      this.drawFriends();
      this.drawRecoverPhase();

      // Draw labels
      this.drawLabels();

      // Draw "time passes" indicator
      if (this.phase === 'wait' || (this.phase === 'recover' && this.phaseTime < 500)) {
        this.drawTimeIndicator();
      }
    }

    drawConnections() {
      const { ctx, layout } = this;

      ctx.lineWidth = 2;
      ctx.setLineDash([]);

      // Split phase connections
      const splitAlpha = Math.min(1, this.splitProgress * 3);
      ctx.strokeStyle = this.alphaColor(COLORS.lineActive, splitAlpha * 0.5);

      // File → Encrypt
      if (this.splitProgress > 0) {
        this.drawLine(layout.file, layout.encrypt);
      }

      // Encrypt → Split
      if (this.splitProgress > 0.3) {
        this.drawLine(layout.encrypt, layout.split);
      }

      // Split → Friends
      ctx.strokeStyle = this.alphaColor(COLORS.line, 0.3);
      for (let i = 0; i < 5; i++) {
        if (this.sharePositions[i] > 0) {
          ctx.strokeStyle = this.alphaColor(COLORS.friend[i], 0.4);
          this.drawLine(layout.split, layout.friends[i]);
        }
      }

      // Friends → Combine (only active ones)
      if (this.phase === 'recover' || this.phase === 'combine' || this.phase === 'complete') {
        for (let i = 0; i < 3; i++) {
          const friendIdx = this.activeFriends[i];
          const alpha = this.recoverPositions[i] * 0.6;
          ctx.strokeStyle = this.alphaColor(COLORS.friend[friendIdx], alpha);
          this.drawLine(layout.friends[friendIdx], layout.combine);
        }
      }

      // Combine → Recovered
      if (this.combineProgress > 0) {
        const goldAlpha = this.combineProgress * 0.6;
        ctx.strokeStyle = this.alphaColor(COLORS.gold, goldAlpha);
        ctx.lineWidth = 3;
        this.drawLine(layout.combine, layout.recovered);
      }
    }

    drawLine(from, to) {
      const { ctx } = this;
      ctx.beginPath();
      ctx.moveTo(from.x, from.y);
      ctx.lineTo(to.x, to.y);
      ctx.stroke();
    }

    drawParticles() {
      const { ctx } = this;

      for (const p of this.particles) {
        const x = p.startX + (p.endX - p.startX) * ease.inOut(p.progress);
        const y = p.startY + (p.endY - p.startY) * ease.inOut(p.progress);

        ctx.beginPath();
        ctx.arc(x, y, p.size, 0, Math.PI * 2);
        ctx.fillStyle = this.alphaColor(p.color, p.life * 0.8);
        ctx.fill();
      }
    }

    drawSplitPhase() {
      const { ctx, layout } = this;
      const r = layout.nodeRadius;

      // File icon
      const fileAlpha = this.splitProgress > 0.5 ? 0.5 : 1;
      this.drawNode(layout.file.x, layout.file.y, r, COLORS.file, COLORS.fileLight, fileAlpha);
      this.drawFileIcon(layout.file.x, layout.file.y, r * 0.5);

      // Encrypt node
      if (this.splitProgress > 0.2) {
        const encAlpha = this.splitProgress > 0.7 ? 0.5 : Math.min(1, (this.splitProgress - 0.2) * 3);
        this.drawNode(layout.encrypt.x, layout.encrypt.y, r * 0.85, COLORS.encrypt, COLORS.encrypt, encAlpha);
        this.drawLockIcon(layout.encrypt.x, layout.encrypt.y, r * 0.4);
      }

      // Split node
      if (this.splitProgress > 0.5) {
        const splitAlpha = Math.min(1, (this.splitProgress - 0.5) * 3);
        this.drawNode(layout.split.x, layout.split.y, r * 0.85, COLORS.split, COLORS.split, splitAlpha);
        this.drawSplitIcon(layout.split.x, layout.split.y, r * 0.4);
      }
    }

    drawFriends() {
      const { ctx, layout } = this;
      const r = layout.smallRadius;

      for (let i = 0; i < 5; i++) {
        const friend = layout.friends[i];
        const hasShare = this.sharePositions[i] >= 1;
        const isActive = this.activeFriends.includes(i);
        const isRecovering = isActive && (this.phase === 'recover' || this.phase === 'combine' || this.phase === 'complete');

        let alpha = 0.3;
        let color = COLORS.friendInactive;

        if (this.sharePositions[i] > 0) {
          alpha = 0.5 + this.sharePositions[i] * 0.5;
          color = COLORS.friend[i];
        }

        if (isRecovering) {
          // Pulse effect for active friends
          const pulse = 1 + Math.sin(this.time * 0.005) * 0.1;
          this.drawNode(friend.x, friend.y, r * pulse, color, color, 1);

          // Glow effect
          ctx.beginPath();
          ctx.arc(friend.x, friend.y, r * 1.5, 0, Math.PI * 2);
          ctx.fillStyle = this.alphaColor(color, 0.2);
          ctx.fill();
        } else {
          this.drawNode(friend.x, friend.y, r, color, color, alpha);
        }

        // Friend initial
        ctx.fillStyle = hasShare ? '#fff' : COLORS.textLight;
        ctx.font = `bold ${r * 0.75}px system-ui, sans-serif`;
        ctx.textAlign = 'center';
        ctx.textBaseline = 'middle';
        ctx.fillText(FRIEND_NAMES[i][0], friend.x, friend.y);

        // Friend name label to the right
        if (hasShare) {
          ctx.fillStyle = COLORS.text;
          ctx.font = '11px system-ui, sans-serif';
          ctx.textAlign = 'left';
          ctx.fillText(FRIEND_NAMES[i], friend.x + r + 6, friend.y + 1);
        }
      }
    }

    drawRecoverPhase() {
      const { ctx, layout } = this;
      const r = layout.nodeRadius;

      // Only show recovery phase elements when appropriate
      if (this.phase !== 'recover' && this.phase !== 'combine' && this.phase !== 'complete') {
        return;
      }

      // Combine node
      const combineAlpha = Math.min(1, this.recoverPositions.reduce((a, b) => a + b, 0) / 2);
      if (combineAlpha > 0) {
        this.drawNode(layout.combine.x, layout.combine.y, r * 0.85, COLORS.gold, COLORS.goldLight, combineAlpha);

        // Draw combining fragments with golden seams
        if (this.combineProgress > 0) {
          this.drawCombiningFragments(layout.combine.x, layout.combine.y, r);
        }
      }

      // Recovered file
      if (this.fileRecovered) {
        const recAlpha = Math.min(1, (this.combineProgress - 0.7) * 4);
        this.drawNode(layout.recovered.x, layout.recovered.y, r, COLORS.recovered, COLORS.recovered, recAlpha);
        this.drawFileIcon(layout.recovered.x, layout.recovered.y, r * 0.5, recAlpha);

        // Success glow
        ctx.beginPath();
        ctx.arc(layout.recovered.x, layout.recovered.y, r * 1.8, 0, Math.PI * 2);
        ctx.fillStyle = this.alphaColor(COLORS.recovered, recAlpha * 0.15);
        ctx.fill();
      }
    }

    drawCombiningFragments(x, y, r) {
      const { ctx } = this;
      const progress = this.combineProgress;

      // Draw 3 fragment pieces coming together
      for (let i = 0; i < 3; i++) {
        const angle = (i / 3) * Math.PI * 2 - Math.PI / 2;
        const dist = r * 0.4 * (1 - progress);
        const fragX = x + Math.cos(angle) * dist;
        const fragY = y + Math.sin(angle) * dist;

        ctx.beginPath();
        ctx.arc(fragX, fragY, r * 0.25, 0, Math.PI * 2);
        ctx.fillStyle = COLORS.friend[this.activeFriends[i]];
        ctx.fill();
      }

      // Golden seams (kintsugi effect)
      if (progress > 0.3) {
        const seamAlpha = (progress - 0.3) / 0.7;
        ctx.strokeStyle = this.alphaColor(COLORS.gold, seamAlpha);
        ctx.lineWidth = 2;

        for (let i = 0; i < 3; i++) {
          const angle1 = (i / 3) * Math.PI * 2 - Math.PI / 2;
          const angle2 = ((i + 1) / 3) * Math.PI * 2 - Math.PI / 2;

          ctx.beginPath();
          ctx.moveTo(x, y);
          ctx.lineTo(x + Math.cos(angle1) * r * 0.3, y + Math.sin(angle1) * r * 0.3);
          ctx.stroke();
        }

        // Golden glow
        ctx.beginPath();
        ctx.arc(x, y, r * 0.5, 0, Math.PI * 2);
        ctx.fillStyle = this.alphaColor(COLORS.goldLight, seamAlpha * 0.3);
        ctx.fill();
      }
    }

    drawLabels() {
      const { ctx, layout } = this;
      const offset = layout.labelOffset;

      ctx.font = '12px system-ui, sans-serif';
      ctx.textAlign = 'center';
      ctx.fillStyle = COLORS.text;

      // Split phase labels
      ctx.fillText('Your File', layout.file.x, layout.file.y + offset);

      if (this.splitProgress > 0.3) {
        ctx.fillText('Encrypt', layout.encrypt.x, layout.encrypt.y + offset);
      }

      if (this.splitProgress > 0.6) {
        ctx.fillText('Split', layout.split.x, layout.split.y + offset);
      }

      // Friends label (names shown next to each friend now)

      // Recovery phase labels
      if (this.phase === 'recover' || this.phase === 'combine' || this.phase === 'complete') {
        if (this.recoverPositions[0] > 0.3) {
          ctx.fillText('Combine', layout.combine.x, layout.combine.y + offset);
        }

        if (this.fileRecovered) {
          ctx.fillStyle = COLORS.recovered;
          ctx.font = 'bold 12px system-ui, sans-serif';
          ctx.fillText('Recovered!', layout.recovered.x, layout.recovered.y + offset);
        }

        // Show which friends are helping (by name)
        const activeNames = this.activeFriends.map(i => FRIEND_NAMES[i]);
        let namesList;
        if (activeNames.length === 3) {
          namesList = `${activeNames[0]}, ${activeNames[1]} & ${activeNames[2]}`;
        } else {
          namesList = activeNames.join(', ');
        }
        ctx.fillStyle = COLORS.textLight;
        ctx.font = '11px system-ui, sans-serif';
        ctx.fillText(namesList, layout.combine.x, layout.combine.y + offset + 14);
      }
    }

    drawTimeIndicator() {
      const { ctx } = this;
      const x = this.width / 2;
      const y = this.height - 20;

      ctx.fillStyle = COLORS.textLight;
      ctx.font = 'italic 13px system-ui, sans-serif';
      ctx.textAlign = 'center';
      ctx.fillText('Later: Recovery needed...', x, y);
    }

    drawNode(x, y, r, color, lightColor, alpha = 1) {
      const { ctx } = this;

      ctx.save();
      ctx.globalAlpha = alpha;

      // Shadow
      ctx.beginPath();
      ctx.arc(x + 2, y + 2, r, 0, Math.PI * 2);
      ctx.fillStyle = 'rgba(0,0,0,0.1)';
      ctx.fill();

      // Gradient fill
      const gradient = ctx.createRadialGradient(x - r * 0.3, y - r * 0.3, 0, x, y, r);
      gradient.addColorStop(0, lightColor);
      gradient.addColorStop(1, color);

      ctx.beginPath();
      ctx.arc(x, y, r, 0, Math.PI * 2);
      ctx.fillStyle = gradient;
      ctx.fill();

      ctx.restore();
    }

    drawFileIcon(x, y, size, alpha = 1) {
      const { ctx } = this;
      ctx.save();
      ctx.globalAlpha = alpha;
      ctx.strokeStyle = '#fff';
      ctx.lineWidth = 1.5;
      ctx.lineCap = 'round';
      ctx.lineJoin = 'round';

      // Simple file shape
      const w = size * 0.7;
      const h = size;
      ctx.beginPath();
      ctx.moveTo(x - w / 2, y - h / 2);
      ctx.lineTo(x + w / 2 - w * 0.3, y - h / 2);
      ctx.lineTo(x + w / 2, y - h / 2 + w * 0.3);
      ctx.lineTo(x + w / 2, y + h / 2);
      ctx.lineTo(x - w / 2, y + h / 2);
      ctx.closePath();
      ctx.stroke();

      // Fold corner
      ctx.beginPath();
      ctx.moveTo(x + w / 2 - w * 0.3, y - h / 2);
      ctx.lineTo(x + w / 2 - w * 0.3, y - h / 2 + w * 0.3);
      ctx.lineTo(x + w / 2, y - h / 2 + w * 0.3);
      ctx.stroke();

      ctx.restore();
    }

    drawLockIcon(x, y, size) {
      const { ctx } = this;
      ctx.strokeStyle = '#fff';
      ctx.fillStyle = '#fff';
      ctx.lineWidth = 1.5;

      // Lock body
      const w = size * 0.8;
      const h = size * 0.6;
      ctx.fillRect(x - w / 2, y - h / 4, w, h);

      // Lock shackle
      ctx.beginPath();
      ctx.arc(x, y - h / 4, w * 0.35, Math.PI, 0);
      ctx.stroke();
    }

    drawSplitIcon(x, y, size) {
      const { ctx } = this;
      ctx.strokeStyle = '#fff';
      ctx.lineWidth = 1.5;
      ctx.lineCap = 'round';

      // Three diverging lines
      ctx.beginPath();
      ctx.moveTo(x - size * 0.4, y);
      ctx.lineTo(x + size * 0.2, y - size * 0.4);
      ctx.moveTo(x - size * 0.4, y);
      ctx.lineTo(x + size * 0.4, y);
      ctx.moveTo(x - size * 0.4, y);
      ctx.lineTo(x + size * 0.2, y + size * 0.4);
      ctx.stroke();
    }

    alphaColor(color, alpha) {
      // Handle hex colors
      if (color.startsWith('#')) {
        const r = parseInt(color.slice(1, 3), 16);
        const g = parseInt(color.slice(3, 5), 16);
        const b = parseInt(color.slice(5, 7), 16);
        return `rgba(${r},${g},${b},${alpha})`;
      }
      return color;
    }

    animate(time) {
      if (!this.running) return;

      const dt = this.lastTime ? time - this.lastTime : 16;
      this.lastTime = time;

      this.update(dt);
      this.draw();

      requestAnimationFrame((t) => this.animate(t));
    }

    start() {
      if (this.running) return;
      this.running = true;
      requestAnimationFrame((t) => this.animate(t));
    }

    stop() {
      this.running = false;
    }
  }

  // Initialize
  function init() {
    if (prefersReducedMotion) return;

    const canvas = document.getElementById('dataflow-canvas');
    if (!canvas) return;

    const dataflow = new DataFlowCanvas(canvas);
    dataflow.start();
    canvas._dataflow = dataflow;
  }

  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }
})();
