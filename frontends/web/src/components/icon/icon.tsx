/**
 * Copyright 2018 Shift Devices AG
 * Copyright 2021 Shift Crypto AG
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { h, JSX } from 'preact';
import alert from './assets/icons/alert-triangle.svg';
import BB02Stylized from '../../assets/device/bitbox02-stylized-reflection.png';
import info from './assets/icons/info.svg';
import arrowDownSVG from './assets/icons/arrow-down-active.svg';
import checkSVG from './assets/icons/check.svg';
import checkedSVG from './assets/icons/checked.svg';
import cancelSVG from './assets/icons/cancel.svg';
import copySVG from './assets/icons/copy.svg';
import * as style from './icon.css';

export const ExpandOpen = (): JSX.Element => (
    <svg
        className={style.expandIcon}
        xmlns="http://www.w3.org/2000/svg"
        width="24"
        height="24"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        <line x1="11" y1="8" x2="11" y2="14"></line>
        <line x1="8" y1="11" x2="14" y2="11"></line>
    </svg>
);

export const ExpandClose = (): JSX.Element => (
    <svg
        className={style.expandIcon}
        xmlns="http://www.w3.org/2000/svg"
        width="24"
        height="24"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round">
        <circle cx="11" cy="11" r="8"></circle>
        <line x1="21" y1="21" x2="16.65" y2="16.65"></line>
        <line x1="8" y1="11" x2="14" y2="11"></line>
    </svg>
);

interface ExpandIconProps {
    expand: boolean;
}

export const ExpandIcon = ({
    expand = true,
}: ExpandIconProps): JSX.Element => (
    expand ? <ExpandOpen /> : <ExpandClose />
);

type IconProps = JSX.IntrinsicElements['img'];

export const Alert = (props: IconProps) => (<img src={alert} draggable={false} {...props} />);
export const Info = (props: IconProps) => (<img src={info} draggable={false} {...props} />);
export const BitBox02Stylized = (props: IconProps) => (<img src={BB02Stylized} draggable={false} {...props} />);
export const ArrowDown = (props: IconProps) => (<img src={arrowDownSVG} draggable={false} {...props} />);
export const Check = (props: IconProps) => (<img src={checkSVG} draggable={false} {...props} />);
export const Checked = (props: IconProps) => (<img src={checkedSVG} draggable={false} {...props} />);
export const Cancel = (props: IconProps) => (<img src={cancelSVG} draggable={false} {...props} />);
export const Copy = (props: IconProps) => (<img src={copySVG} draggable={false} {...props} />);
