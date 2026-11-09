using System;
using System.Collections.Generic;
using System.Globalization;
using System.Windows;
using System.Windows.Data;
using SpecDesignerCommon.Helpers;

namespace SpecDesigner.SpecEditor.Helpers
{
	// Token: 0x02000049 RID: 73
	internal class MultiVisibilityConverter : IMultiValueConverter
	{
		// Token: 0x060001CA RID: 458 RVA: 0x0000C4E8 File Offset: 0x0000A6E8
		public object Convert(object[] values, Type targetType, object parameter, CultureInfo culture)
		{
			string text = (string)parameter;
			string text2 = "";
			ComponentType componentType = ComponentType.Unknown;
			ComponentType componentType2 = ComponentType.Unknown;
			if (values[0] == null || values[0] == DependencyProperty.UnsetValue)
			{
				return Visibility.Collapsed;
			}
			if (values[1] != DependencyProperty.UnsetValue)
			{
				if (values[1] is ComponentType)
				{
					componentType = (ComponentType)values[1];
				}
				else
				{
					componentType = (ComponentType)Enum.Parse(typeof(ComponentType), (string)values[1]);
				}
			}
			if (values[2] != DependencyProperty.UnsetValue)
			{
				if (values[2] is ComponentType)
				{
					componentType2 = (ComponentType)values[2];
				}
				else
				{
					componentType2 = (ComponentType)Enum.Parse(typeof(ComponentType), (string)values[2]);
				}
			}
			if (values[4] != DependencyProperty.UnsetValue)
			{
				text2 = (string)values[4];
			}
			string text3 = "";
			switch (componentType)
			{
			case ComponentType.Label:
			case ComponentType.Edit:
			case ComponentType.ComboBox:
			case ComponentType.TextEdit:
			case ComponentType.ButtonEdit:
			case ComponentType.DateEdit:
			case ComponentType.CheckBox:
			case ComponentType.FFLabel:
				if (text2 == "reference")
				{
					SpecPropertyEditor.ComponentPropertyVisibilityMap["Reference"].TryGetValue(text, out text3);
				}
				else
				{
					Dictionary<string, string> dictionary;
					SpecPropertyEditor.ComponentPropertyVisibilityMap.TryGetValue(componentType.ToString(), out dictionary);
					if (dictionary != null)
					{
						dictionary.TryGetValue(text, out text3);
					}
				}
				if (componentType == ComponentType.CheckBox && componentType2 == ComponentType.Table && text == "text")
				{
					text3 = "L";
				}
				if (componentType == ComponentType.Edit && componentType2 == ComponentType.Table && text == "aggregate" && text2 != "reference")
				{
					text3 = "S";
				}
				if (componentType2 == ComponentType.Table && componentType != ComponentType.Edit && componentType != ComponentType.ButtonEdit && componentType != ComponentType.FFLabel && text == "ProgRelFieldGroup")
				{
					text3 = "L";
				}
				break;
			case ComponentType.Button:
				if (text2 == "button_qrystr")
				{
					SpecPropertyEditor.ComponentPropertyVisibilityMap["ProgRelField"].TryGetValue(text, out text3);
				}
				break;
			}
			if ((componentType2 == ComponentType.Tree || componentType2 == ComponentType.Table) && componentType != ComponentType.Phantom && componentType != ComponentType.CheckBox && text == "text")
			{
				text3 = "H";
			}
			string text4;
			if (!string.IsNullOrEmpty(text3) && (text4 = text3) != null)
			{
				if (text4 == "S")
				{
					return Visibility.Visible;
				}
				if (text4 == "H")
				{
					return Visibility.Collapsed;
				}
				if (text4 == "L")
				{
					return ((bool)values[3]) ? Visibility.Visible : Visibility.Collapsed;
				}
			}
			return Visibility.Visible;
		}

		// Token: 0x060001CB RID: 459 RVA: 0x0000C762 File Offset: 0x0000A962
		public object[] ConvertBack(object value, Type[] targetTypes, object parameter, CultureInfo culture)
		{
			throw new NotImplementedException();
		}
	}
}
