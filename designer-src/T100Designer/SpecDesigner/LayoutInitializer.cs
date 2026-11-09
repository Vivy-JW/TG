using System;
using System.Linq;
using System.Windows;
using SpecDesigner.ViewModels;
using Xceed.Wpf.AvalonDock.Layout;

namespace SpecDesigner
{
	// Token: 0x02000009 RID: 9
	internal class LayoutInitializer : ILayoutUpdateStrategy
	{
		// Token: 0x06000054 RID: 84 RVA: 0x00002F68 File Offset: 0x00001168
		public bool BeforeInsertAnchorable(LayoutRoot layout, LayoutAnchorable anchorableToShow, ILayoutContainer destinationContainer)
		{
			if (destinationContainer != null && destinationContainer.FindParent<LayoutFloatingWindow>() != null)
			{
				return false;
			}
			if (anchorableToShow.Content is PaneViewModel)
			{
				PaneViewModel svm = anchorableToShow.Content as PaneViewModel;
				LayoutAnchorablePane layoutAnchorablePane = layout.Descendents().OfType<LayoutAnchorablePane>().FirstOrDefault<LayoutAnchorablePane>((LayoutAnchorablePane d) => d.Name == svm.ContentId);
				if (layoutAnchorablePane != null)
				{
					if (layoutAnchorablePane.ChildrenCount == 0)
					{
						layoutAnchorablePane.DockWidth = new GridLength(100.0);
					}
					layoutAnchorablePane.Children.Add(anchorableToShow);
					return true;
				}
			}
			LayoutAnchorablePane layoutAnchorablePane2 = layout.Descendents().OfType<LayoutAnchorablePane>().FirstOrDefault<LayoutAnchorablePane>((LayoutAnchorablePane d) => d.Name == "InfomationPane");
			if (layoutAnchorablePane2 != null)
			{
				layoutAnchorablePane2.Children.Add(anchorableToShow);
				return true;
			}
			return false;
		}

		// Token: 0x06000055 RID: 85 RVA: 0x00003030 File Offset: 0x00001230
		public void AfterInsertAnchorable(LayoutRoot layout, LayoutAnchorable anchorableShown)
		{
		}

		// Token: 0x06000056 RID: 86 RVA: 0x00003032 File Offset: 0x00001232
		public bool BeforeInsertDocument(LayoutRoot layout, LayoutDocument anchorableToShow, ILayoutContainer destinationContainer)
		{
			return false;
		}

		// Token: 0x06000057 RID: 87 RVA: 0x00003035 File Offset: 0x00001235
		public void AfterInsertDocument(LayoutRoot layout, LayoutDocument anchorableShown)
		{
		}
	}
}
