using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;

namespace SpecDesigner.CodeEditWindow.Extension
{
	// Token: 0x02000041 RID: 65
	public static class DragDropExtension
	{
		// Token: 0x060002D4 RID: 724 RVA: 0x00018A27 File Offset: 0x00016C27
		public static bool GetScrollOnDragDrop(DependencyObject element)
		{
			if (element == null)
			{
				throw new ArgumentNullException("element");
			}
			return (bool)element.GetValue(DragDropExtension.ScrollOnDragDropProperty);
		}

		// Token: 0x060002D5 RID: 725 RVA: 0x00018A47 File Offset: 0x00016C47
		public static void SetScrollOnDragDrop(DependencyObject element, bool value)
		{
			if (element == null)
			{
				throw new ArgumentNullException("element");
			}
			element.SetValue(DragDropExtension.ScrollOnDragDropProperty, value);
		}

		// Token: 0x060002D6 RID: 726 RVA: 0x00018A68 File Offset: 0x00016C68
		private static void HandleScrollOnDragDropChanged(DependencyObject d, DependencyPropertyChangedEventArgs e)
		{
			FrameworkElement frameworkElement = d as FrameworkElement;
			if (d == null)
			{
				return;
			}
			DragDropExtension.Unsubscribe(frameworkElement);
			if (true.Equals(e.NewValue))
			{
				DragDropExtension.Subscribe(frameworkElement);
			}
		}

		// Token: 0x060002D7 RID: 727 RVA: 0x00018A9E File Offset: 0x00016C9E
		private static void Subscribe(FrameworkElement container)
		{
			container.PreviewDragOver += DragDropExtension.OnContainerPreviewDragOver;
		}

		// Token: 0x060002D8 RID: 728 RVA: 0x00018AB4 File Offset: 0x00016CB4
		private static void OnContainerPreviewDragOver(object sender, DragEventArgs e)
		{
			FrameworkElement frameworkElement = sender as FrameworkElement;
			if (frameworkElement == null)
			{
				return;
			}
			ScrollViewer firstVisualChild = DragDropExtension.GetFirstVisualChild<ScrollViewer>(frameworkElement);
			if (firstVisualChild == null)
			{
				return;
			}
			double num = 60.0;
			double y = e.GetPosition(frameworkElement).Y;
			double num2 = 20.0;
			if (y < num)
			{
				firstVisualChild.ScrollToVerticalOffset(firstVisualChild.VerticalOffset - num2);
				return;
			}
			if (y > frameworkElement.ActualHeight - num)
			{
				firstVisualChild.ScrollToVerticalOffset(firstVisualChild.VerticalOffset + num2);
			}
		}

		// Token: 0x060002D9 RID: 729 RVA: 0x00018B2B File Offset: 0x00016D2B
		private static void Unsubscribe(FrameworkElement container)
		{
			container.PreviewDragOver -= DragDropExtension.OnContainerPreviewDragOver;
		}

		// Token: 0x060002DA RID: 730 RVA: 0x00018B40 File Offset: 0x00016D40
		public static T GetFirstVisualChild<T>(DependencyObject depObj) where T : DependencyObject
		{
			if (depObj != null)
			{
				for (int i = 0; i < VisualTreeHelper.GetChildrenCount(depObj); i++)
				{
					DependencyObject child = VisualTreeHelper.GetChild(depObj, i);
					if (child != null && child is T)
					{
						return (T)((object)child);
					}
					T firstVisualChild = DragDropExtension.GetFirstVisualChild<T>(child);
					if (firstVisualChild != null)
					{
						return firstVisualChild;
					}
				}
			}
			return default(T);
		}

		// Token: 0x0400013D RID: 317
		public static readonly DependencyProperty ScrollOnDragDropProperty = DependencyProperty.RegisterAttached("ScrollOnDragDrop", typeof(bool), typeof(DragDropExtension), new PropertyMetadata(false, new PropertyChangedCallback(DragDropExtension.HandleScrollOnDragDropChanged)));
	}
}
