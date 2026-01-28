// Copyright (C) The Arvados Authors. All rights reserved.
//
// SPDX-License-Identifier: AGPL-3.0

import React, { useRef, useLayoutEffect, useState } from 'react'
import { CircularProgress } from '@mui/material'
import { WithStyles, withStyles } from '@mui/styles'
import { CustomStyleRulesCallback } from 'common/custom-theme'

type CssRules = 'container'

const styles: CustomStyleRulesCallback<CssRules> = () => ({
	container: {
		display: 'flex',
		alignItems: 'center',
		justifyContent: 'center',
	},
})

type CircularSuspenseProps = {
	element: React.ReactNode
	showElement: boolean
}

/**
 * A loading component that replaces an element with a circular progress indicator.
 *
 * The component measures the dimensions of the provided element and displays a
 * CircularProgress spinner in a container that matches those exact dimensions.
 * No layout shift should occur when switching between the element and the spinner.
 *
 * @param element - The React element to display when showElement is true
 * @param showElement - Whether to show the element (true) or the loading spinner (false)
 * @returns The element or a loading spinner in a container matching the element's size
 */
export const CircularSuspense = withStyles(styles)(({
	element,
	showElement,
	classes,
}: CircularSuspenseProps & WithStyles<CssRules>) => {
	const elementRef = useRef<HTMLDivElement>(null)
	const [dimensions, setDimensions] = useState<{ width: number; height: number } | null>(null)

	useLayoutEffect(() => {
		if (elementRef.current && !dimensions) {
			const rect = elementRef.current.getBoundingClientRect()
			setDimensions({
				width: rect.width,
				height: rect.height,
			})
		}
	}, [showElement, element, dimensions])

	if (showElement) {
		return <div ref={elementRef}>{element}</div>
	}

	if (!dimensions) {
		return (
			<div ref={elementRef} style={{ visibility: 'hidden', position: 'absolute' }}>
				{element}
			</div>
		)
	}

	const maxSize = Math.min(dimensions.width, dimensions.height) * 0.8

	return (
		<div
			className={classes.container}
			style={{
				width: dimensions.width,
				height: dimensions.height,
			}}
		>
			<CircularProgress size={maxSize} />
		</div>
	)
})
