using System;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Media;

namespace SpecDesigner.SpecEditor.Views
{
	// Token: 0x02000034 RID: 52
	public class GridLineHelper
	{
		// Token: 0x06000153 RID: 339 RVA: 0x0000A404 File Offset: 0x00008604
		public static bool GetShowGridLine(DependencyObject obj)
		{
			return (bool)obj.GetValue(GridLineHelper.ShowGridLineProperty);
		}

		// Token: 0x06000154 RID: 340 RVA: 0x0000A416 File Offset: 0x00008616
		public static void SetShowGridLine(DependencyObject obj, bool value)
		{
			obj.SetValue(GridLineHelper.ShowGridLineProperty, value);
		}

		// Token: 0x06000155 RID: 341 RVA: 0x0000A4D0 File Offset: 0x000086D0
		private static void OnShowBorderChanged(DependencyObject sener, DependencyPropertyChangedEventArgs e)
		{
			Grid grid = sener as Grid;
			if (grid == null)
			{
				return;
			}
			if ((bool)e.OldValue)
			{
				grid.Loaded -= delegate(object s, RoutedEventArgs arg)
				{
				};
			}
			if ((bool)e.NewValue)
			{
				grid.Loaded += delegate(object s, RoutedEventArgs arg)
				{
					int count = grid.RowDefinitions.Count;
					int count2 = grid.ColumnDefinitions.Count;
					for (int i = 0; i < count; i++)
					{
						for (int j = 0; j < count2; j++)
						{
							Border border = new Border
							{
								BorderBrush = new SolidColorBrush(Colors.LightGray),
								BorderThickness = new Thickness(1.0)
							};
							Grid.SetRow(border, i);
							Grid.SetColumn(border, j);
							grid.Children.Add(border);
						}
					}
				};
			}
		}

		// Token: 0x040000A0 RID: 160
		public static readonly DependencyProperty ShowGridLineProperty = DependencyProperty.RegisterAttached("ShowGridLine", typeof(bool), typeof(GridLineHelper), new PropertyMetadata(new PropertyChangedCallback(GridLineHelper.OnShowBorderChanged)));
	}
}
