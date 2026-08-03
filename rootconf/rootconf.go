/*
 * Copyright 2024 hopeio. All rights reserved.
 * Licensed under the MIT License that can be found in the LICENSE file.
 * @Created by jyb
 */

// Package rootconf 历史上承载 RootConfig 等根配置类型。
// 类型定义已迁入 initialize 根包，本包仅保留类型别名以兼容既有引用方，
// 新代码请直接使用 github.com/hopeio/initialize 中的同名类型。
package rootconf

import "github.com/hopeio/initialize"

type RootConfig = initialize.RootConfig

// BasicConfig
type BasicConfig = initialize.BasicConfig

type EnvConfig = initialize.EnvConfig
