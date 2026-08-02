/**
 * @mgd/domain-states：面试领域状态枚举的单一共享模块（需求 G1）。
 * 页面层必须从此处导入状态标识，禁止书写状态字面量。
 */

export {
  ACCOMMODATION_KEYS,
  defaultAccommodations,
  isAccommodationKey,
  type AccommodationKey,
} from './accommodations.ts';

export {
  ORG_ASSIGNMENT_PROGRESSES,
  ORG_MIN_COHORT_SIZE,
  type OrgAssignmentProgress,
} from './org.ts';

export {
  isProjectStatus,
  isTerminalProjectStatus,
  PROJECT_STATUSES,
  projectStatusTone,
  type ProjectStatus,
  type ResultTone,
} from './project.ts';

export {
  isAvatarBillable,
  isSessionStatus,
  isSystemPause,
  isTimerRunning,
  SESSION_STATUSES,
  type SessionStatus,
} from './session.ts';
