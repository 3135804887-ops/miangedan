/** SCR-08/09 房间组件测试：红线文案与降级覆盖层。 */

import { ToastProvider } from '@mgd/ui';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';

import { RoomView } from '../src/components/room-view.tsx';

const labels = {
  title: '实时面试',
  round: '第 {n} 轮',
  live: '进行中',
  paused: '已暂停',
  reconnecting: '重连中',
  timer: '本轮用时',
  network: '网络',
  subtitles: '字幕',
  on: '开',
  off: '关',
  exit: '退出面试',
  exitDialogTitle: '退出当前面试？',
  exitDialogBody: '退出将标记本轮为评估未完成（不是失败），并按实际使用结算。',
  question: '面试官问题',
  questionHint: '问题固定显示，可回看',
  candidateTranscript: '你的回答（可修订）',
  revisionHint: '进入下一主问题后本轮转写将冻结，评分使用修订文本。',
  revise: '修订',
  revised: '已修订',
  frozen: '已冻结',
  dynamicFollowup: '动态追问',
  tools: '岗位工具',
  noToolsHint: '本轮未配置工具',
  controls: '控制',
  stopAvatar: '停止数字人',
  textAnswer: '输入文字回答……',
  submitAnswer: '提交',
  cameraLabel: '摄像头',
  micLabel: '麦克风',
  turnProgress: '覆盖点进度',
  systemPaused: '系统故障，计时已暂停',
  systemPausedHint: '正在自动恢复；此段时间不计费、不影响评分。',
  reconnectCountdown: '请在 {seconds} 秒内重新连接',
  reconnectHint: '刷新/断网后可在倒计时内恢复到最后已确认回合，不扣时间、不判失败。',
  downgradeTitle: '数字人音视频暂时无法恢复',
  downgradeBody: '是否改为文字面试继续？',
  downgradeAccept: '改为文字面试',
  downgradeDecline: '结束并保留额度',
  downgradeDeclineHint: '结束将标记评估未完成（不是失败）。',
  authPaused: '登录状态需要重新确认',
  authPausedHint: '计时已暂停，不会扣时间也不会判失败。',
  reauthenticate: '重新认证',
} as const;

describe('RoomView（SCR-08/09）', () => {
  it('渲染房间布局与红线文案（修订窗口提示）', () => {
    render(
    <ToastProvider>
      <RoomView locale="zh-CN" labels={labels} sessionId="s-0001" />
      </ToastProvider>,
    );
    expect(screen.getByText('面试官问题')).toBeDefined();
    expect(screen.getByText('你的回答（可修订）')).toBeDefined();
    expect(screen.getAllByText(/转写将冻结/).length).toBeGreaterThan(0);
  });

  it('触发降级询问覆盖层：同意后切换文字面试', async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
      <RoomView locale="zh-CN" labels={labels} sessionId="s-0001" />
      </ToastProvider>,
    );
    await user.click(screen.getByText('演示：降级询问'));
    expect(screen.getByText('数字人音视频暂时无法恢复')).toBeDefined();
    await user.click(screen.getByText('改为文字面试'));
    expect(screen.queryByText('数字人音视频暂时无法恢复')).toBeNull();
  });

  it('触发系统暂停覆盖层并恢复', async () => {
    const user = userEvent.setup();
    render(
      <ToastProvider>
      <RoomView locale="zh-CN" labels={labels} sessionId="s-0001" />
      </ToastProvider>,
    );
    await user.click(screen.getByText('演示：触发系统故障暂停'));
    expect(screen.getByText('系统故障，计时已暂停')).toBeDefined();
    await user.click(screen.getByText('我已恢复'));
    expect(screen.queryByText('系统故障，计时已暂停')).toBeNull();
  });
});
