using System;
using System.ComponentModel;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Data;
using System.Windows.Media;

namespace SpecDesigner.CodeEditWindow.View
{
	// Token: 0x02000028 RID: 40
	public class DSCTextColumn : DataGridTextColumn
	{
		// Token: 0x0600016B RID: 363 RVA: 0x0000D594 File Offset: 0x0000B794
		protected override FrameworkElement GenerateElement(DataGridCell cell, object dataItem)
		{
			TextBlock textBlock = base.GenerateElement(cell, dataItem) as TextBlock;
			Binding binding = this.Binding as Binding;
			Binding binding2 = new Binding
			{
				Path = binding.Path,
				Source = dataItem
			};
			textBlock.SetBinding(TextBlock.TextProperty, binding2);
			string path = ((Binding)this.Binding).Path.Path;
			IDataErrorInfo dataErrorInfo = dataItem as IDataErrorInfo;
			if (dataErrorInfo != null && !string.IsNullOrWhiteSpace(dataErrorInfo[path]))
			{
				textBlock.Background = Brushes.Red;
				textBlock.ToolTip = new TextBlock
				{
					Text = dataErrorInfo[path]
				};
			}
			return textBlock;
		}
	}
}
