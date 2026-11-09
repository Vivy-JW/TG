using System;
using System.Collections.Generic;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Media;
using System.Windows.Threading;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x02000010 RID: 16
	public class DataGridHelper
	{
		// Token: 0x0600006B RID: 107 RVA: 0x0000397C File Offset: 0x00001B7C
		public static DataGridCell GetCell(DataGrid dataGrid, int row, int column)
		{
			DataGridRow row2 = DataGridHelper.GetRow(dataGrid, row);
			if (row2 != null)
			{
				DataGridCellsPresenter visualChild = DataGridHelper.GetVisualChild<DataGridCellsPresenter>(row2);
				DataGridCell dataGridCell = (DataGridCell)visualChild.ItemContainerGenerator.ContainerFromIndex(column);
				if (dataGridCell == null)
				{
					dataGrid.ScrollIntoView(row2, dataGrid.Columns[column]);
					dataGridCell = (DataGridCell)visualChild.ItemContainerGenerator.ContainerFromIndex(column);
				}
				return dataGridCell;
			}
			return null;
		}

		// Token: 0x0600006C RID: 108 RVA: 0x000039D8 File Offset: 0x00001BD8
		public static DataGridRow GetRow(DataGrid dataGrid, int index)
		{
			DataGridRow dataGridRow = (DataGridRow)dataGrid.ItemContainerGenerator.ContainerFromIndex(index);
			if (dataGridRow == null)
			{
				dataGrid.ScrollIntoView(dataGrid.Items[index]);
				dataGrid.UpdateLayout();
				dataGridRow = (DataGridRow)dataGrid.ItemContainerGenerator.ContainerFromIndex(index);
			}
			return dataGridRow;
		}

		// Token: 0x0600006D RID: 109 RVA: 0x00003A25 File Offset: 0x00001C25
		public static DataGridRowHeader GetRowHeader(DataGrid dataGrid, int index)
		{
			return DataGridHelper.GetRowHeader(DataGridHelper.GetRow(dataGrid, index));
		}

		// Token: 0x0600006E RID: 110 RVA: 0x00003A33 File Offset: 0x00001C33
		public static DataGridRowHeader GetRowHeader(DataGridRow row)
		{
			if (row != null)
			{
				return DataGridHelper.GetVisualChild<DataGridRowHeader>(row);
			}
			return null;
		}

		// Token: 0x0600006F RID: 111 RVA: 0x00003A40 File Offset: 0x00001C40
		public static DataGridColumnHeader GetColumnHeader(DataGrid dataGrid, int index)
		{
			DataGridColumnHeadersPresenter visualChild = DataGridHelper.GetVisualChild<DataGridColumnHeadersPresenter>(dataGrid);
			if (visualChild != null)
			{
				return (DataGridColumnHeader)visualChild.ItemContainerGenerator.ContainerFromIndex(index);
			}
			return null;
		}

		// Token: 0x06000070 RID: 112 RVA: 0x00003A6C File Offset: 0x00001C6C
		public static T GetVisualChild<T>(Visual parent) where T : Visual
		{
			T t = default(T);
			int childrenCount = VisualTreeHelper.GetChildrenCount(parent);
			for (int i = 0; i < childrenCount; i++)
			{
				Visual visual = (Visual)VisualTreeHelper.GetChild(parent, i);
				t = visual as T;
				if (t == null)
				{
					t = DataGridHelper.GetVisualChild<T>(visual);
				}
				if (t != null)
				{
					break;
				}
			}
			return t;
		}

		// Token: 0x06000071 RID: 113 RVA: 0x00003AC8 File Offset: 0x00001CC8
		public static T GetVisualChild<T>(Visual parent, int index) where T : Visual
		{
			T t = default(T);
			int num = 0;
			Queue<Visual> queue = new Queue<Visual>();
			queue.Enqueue(parent);
			while (queue.Count > 0)
			{
				Visual visual = queue.Dequeue();
				t = visual as T;
				if (t != null)
				{
					if (num == index)
					{
						break;
					}
					num++;
				}
				else
				{
					int childrenCount = VisualTreeHelper.GetChildrenCount(visual);
					for (int i = 0; i < childrenCount; i++)
					{
						queue.Enqueue((Visual)VisualTreeHelper.GetChild(visual, i));
					}
				}
			}
			return t;
		}

		// Token: 0x06000072 RID: 114 RVA: 0x00003B4C File Offset: 0x00001D4C
		public static bool VisualChildExists(Visual parent, DependencyObject visualToFind)
		{
			Queue<Visual> queue = new Queue<Visual>();
			queue.Enqueue(parent);
			while (queue.Count > 0)
			{
				Visual visual = queue.Dequeue();
				DependencyObject dependencyObject = visual;
				if (dependencyObject != null)
				{
					if (dependencyObject == visualToFind)
					{
						return true;
					}
				}
				else
				{
					int childrenCount = VisualTreeHelper.GetChildrenCount(visual);
					for (int i = 0; i < childrenCount; i++)
					{
						queue.Enqueue((Visual)VisualTreeHelper.GetChild(visual, i));
					}
				}
			}
			return false;
		}

		// Token: 0x06000073 RID: 115 RVA: 0x00003BB0 File Offset: 0x00001DB0
		public static DependencyObject FindPartByName(DependencyObject ele, string name)
		{
			if (ele == null)
			{
				return null;
			}
			if (name.Equals(ele.GetValue(FrameworkElement.NameProperty)))
			{
				return ele;
			}
			int childrenCount = VisualTreeHelper.GetChildrenCount(ele);
			for (int i = 0; i < childrenCount; i++)
			{
				DependencyObject child = VisualTreeHelper.GetChild(ele, i);
				DependencyObject dependencyObject;
				if ((dependencyObject = DataGridHelper.FindPartByName(child, name)) != null)
				{
					return dependencyObject;
				}
			}
			return null;
		}

		// Token: 0x06000074 RID: 116 RVA: 0x00003C00 File Offset: 0x00001E00
		public static T FindVisualParent<T>(UIElement element) where T : UIElement
		{
			for (UIElement uielement = element; uielement != null; uielement = VisualTreeHelper.GetParent(uielement) as UIElement)
			{
				T t = uielement as T;
				if (t != null)
				{
					return t;
				}
			}
			return default(T);
		}

		// Token: 0x06000075 RID: 117 RVA: 0x00003C58 File Offset: 0x00001E58
		public static void WaitTillQueueItemsProcessed()
		{
			DispatcherFrame frame = new DispatcherFrame();
			Dispatcher.CurrentDispatcher.BeginInvoke(DispatcherPriority.ApplicationIdle, new DispatcherOperationCallback(delegate(object arg)
			{
				frame.Continue = false;
				return null;
			}), null);
			Dispatcher.PushFrame(frame);
		}
	}
}
