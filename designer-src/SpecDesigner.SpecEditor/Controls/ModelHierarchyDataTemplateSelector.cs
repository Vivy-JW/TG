using System;
using System.Windows;
using System.Windows.Controls;

namespace SpecDesigner.SpecEditor.Controls
{
	// Token: 0x0200002D RID: 45
	public class ModelHierarchyDataTemplateSelector : DataTemplateSelector
	{
		// Token: 0x0600012F RID: 303 RVA: 0x00009C10 File Offset: 0x00007E10
		public override DataTemplate SelectTemplate(object item, DependencyObject container)
		{
			DataTemplate dataTemplate = null;
			FrameworkElement frameworkElement = container as FrameworkElement;
			if (frameworkElement != null && item != null)
			{
				if (item is TblModel)
				{
					dataTemplate = frameworkElement.FindResource("TblModelTemplate") as DataTemplate;
				}
				else if (item is SRModel)
				{
					dataTemplate = frameworkElement.FindResource("SRModelTemplate") as DataTemplate;
				}
			}
			return dataTemplate;
		}
	}
}
