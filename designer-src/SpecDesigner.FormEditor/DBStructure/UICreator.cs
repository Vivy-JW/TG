using System;
using System.Collections.Generic;
using System.Linq;
using System.Windows;
using SpecDesigner.FormEditor.Helpers;
using SpecDesignerCommon;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor.DBStructure
{
	// Token: 0x0200001F RID: 31
	public class UICreator
	{
		// Token: 0x060000FC RID: 252 RVA: 0x00005AE0 File Offset: 0x00003CE0
		internal static void Create(FrameworkElement LayoutRoot, ContainerViewModel containerViewModel, IEnumerable<PrepareAddColumn> fields, PackageKey ProgramKey)
		{
			List<XmlElement> list = new List<XmlElement>();
			string type;
			if ((type = containerViewModel.Container.Type) != null)
			{
				if (type == "None")
				{
					list = UICreator.CreateNoneContainerWidget(containerViewModel, fields, ProgramKey);
					AddWidgetAdornerHelper.Show(LayoutRoot, list, false, SpecNodeType.NONE);
					return;
				}
				if (type == "Grid")
				{
					list = UICreator.CreateGridContainer(containerViewModel, fields, ProgramKey);
					AddWidgetAdornerHelper.Show(LayoutRoot, list, false, SpecNodeType.NONE);
					return;
				}
				if (type == "Group")
				{
					list = UICreator.CreateGroupContainer(containerViewModel, fields, ProgramKey);
					AddWidgetAdornerHelper.Show(LayoutRoot, list, false, SpecNodeType.NONE);
					return;
				}
				if (type == "ScrollGrid")
				{
					list = UICreator.CreateScrollGridContainer(containerViewModel, fields, ProgramKey);
					AddWidgetAdornerHelper.Show(LayoutRoot, list, false, SpecNodeType.NONE);
					return;
				}
				if (type == "Table")
				{
					list = UICreator.CreateTableContainer(containerViewModel, fields, ProgramKey);
					AddWidgetAdornerHelper.Show(LayoutRoot, list, false, SpecNodeType.NONE);
					return;
				}
				if (!(type == "Tree"))
				{
					return;
				}
				list = UICreator.CreateTreeContainer(containerViewModel, fields, ProgramKey);
				AddWidgetAdornerHelper.Show(LayoutRoot, list, false, SpecNodeType.NONE);
			}
		}

		// Token: 0x060000FD RID: 253 RVA: 0x00005BC5 File Offset: 0x00003DC5
		private static string GetNewNameFromColumn(PackageKey key, PrepareAddColumn field)
		{
			if (!string.IsNullOrEmpty(field.Column))
			{
				return ComponentFactory.GetNewName(field.Column, key);
			}
			return ComponentFactory.GetNewName(field.Widget, key);
		}

		// Token: 0x060000FE RID: 254 RVA: 0x00005BF0 File Offset: 0x00003DF0
		private static string GetNewNameFromColumnForDetail(PackageKey key, PrepareAddColumn field)
		{
			string text = ((!string.IsNullOrEmpty(field.Column)) ? field.Column : field.Widget);
			if (SettingManager.Get().GetTzpManger(key).SpecificationInfo.GetCodeTemplate().Equals("Q", StringComparison.CurrentCultureIgnoreCase))
			{
				text = string.Format("{0}_{1}", "b", text);
			}
			return ComponentFactory.GetNewName(text, key);
		}

		// Token: 0x060000FF RID: 255 RVA: 0x00005C54 File Offset: 0x00003E54
		private static XmlElement CreateFieldComponent(PrepareAddColumn field, string newName, PackageKey programKey)
		{
			ComponentType componentType = ComponentType.Edit;
			Enum.TryParse<ComponentType>(field.Widget, true, out componentType);
			XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(programKey, componentType, newName);
			if (xmlElement.GetAttribute("fieldType") != null)
			{
				xmlElement.SetAttribute("sqlTabName", field.Table);
				xmlElement.SetAttribute("colName", field.Column);
				xmlElement.SetAttribute("fieldType", "COLUMN_LIKE");
			}
			return xmlElement;
		}

		// Token: 0x06000100 RID: 256 RVA: 0x00005CBC File Offset: 0x00003EBC
		private static List<XmlElement> CreateNoneContainerWidget(ContainerViewModel containerViewModel, IEnumerable<PrepareAddColumn> fields, PackageKey ProgramKey)
		{
			UICreator.Named.Clear();
			List<XmlElement> list = new List<XmlElement>();
			List<PrepareAddColumn> list2 = fields.ToList<PrepareAddColumn>();
			int num = 0;
			foreach (PrepareAddColumn prepareAddColumn in list2)
			{
				string text = UICreator.GetNewNameFromColumn(ProgramKey, prepareAddColumn);
				if (text.Length != 0)
				{
					XmlElement xmlElement = UICreator.CreateFieldComponent(prepareAddColumn, text, ProgramKey);
					xmlElement.TabIndex += num;
					num++;
					xmlElement.GridWidth = Math.Min(containerViewModel.MaximumWidthIntValue, prepareAddColumn.WidthIntValue);
					text = UICreator.GetCurrentNewName(ProgramKey, ComponentType.Edit, prepareAddColumn);
					string text2 = string.Format("{0}.{1}", prepareAddColumn.Table, prepareAddColumn.Column);
					if (text == text2)
					{
						xmlElement.SetAttribute("fieldType", "TABLE_COLUMN");
					}
					xmlElement.SetAttribute("name", text);
					list.Add(xmlElement);
				}
			}
			int num2 = list2.Count<PrepareAddColumn>();
			int numberOfFieldsIntValue = containerViewModel.NumberOfFieldsIntValue;
			if (num2 % numberOfFieldsIntValue <= 0)
			{
				int num3 = num2 / numberOfFieldsIntValue;
			}
			else
			{
				int num4 = num2 / numberOfFieldsIntValue;
			}
			List<object>[] array = new List<object>[numberOfFieldsIntValue];
			for (int i = 0; i < num2; i++)
			{
				if (array[i % numberOfFieldsIntValue] == null)
				{
					array[i % numberOfFieldsIntValue] = new List<object>();
				}
				array[i % numberOfFieldsIntValue].Add(null);
			}
			int num5 = 0;
			int num6 = 10;
			int num7 = 1;
			int num8 = 12;
			int num9 = 0;
			int num10 = 1;
			List<XmlElement> list3 = new List<XmlElement>();
			int num11 = 0;
			while (num11 < array.Length && array[num11] != null)
			{
				for (int j = 0; j < array[num11].Count; j++)
				{
					if (list2.Count > num5)
					{
						PrepareAddColumn prepareAddColumn2 = list2[num5];
						XmlElement xmlElement2 = list[num5];
						if (!string.IsNullOrEmpty(prepareAddColumn2.Label))
						{
							string currentNewName = UICreator.GetCurrentNewName(ProgramKey, ComponentType.Label, prepareAddColumn2);
							XmlElement xmlElement3 = ComponentFactory.CreateEmptyComponent(ProgramKey, ComponentType.Label, currentNewName);
							xmlElement3.GridWidth = num6;
							xmlElement3.GridX = num7;
							xmlElement3.GridY = num10;
							xmlElement3.LocalString = prepareAddColumn2.Label;
							list3.Add(xmlElement3);
							xmlElement3.BindElement = xmlElement2;
							xmlElement2.BindElement = xmlElement3;
							UICreator.GetSpecificationInfo(ProgramKey).AddSpecBinding(xmlElement3, xmlElement2);
						}
						xmlElement2.GridX = num8;
						xmlElement2.GridY = num10;
						num9 = Math.Max(num9, Math.Min(containerViewModel.MaximumWidthIntValue, prepareAddColumn2.WidthIntValue));
						num10 += xmlElement2.GridHeight;
						num5++;
					}
				}
				num7 += num9 + 11 + 1;
				num8 += num9 + 11 + 1;
				num9 = 0;
				num10 = 1;
				num11++;
			}
			list.AddRange(list3);
			return list;
		}

		// Token: 0x06000101 RID: 257 RVA: 0x00005F88 File Offset: 0x00004188
		private static List<XmlElement> CreateContainerWithFileds(ContainerViewModel containerViewModel, IEnumerable<PrepareAddColumn> fields, ComponentType containerType, PackageKey ProgramKey)
		{
			List<XmlElement> list = UICreator.CreateNoneContainerWidget(containerViewModel, fields, ProgramKey);
			XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(ProgramKey, containerType, null);
			foreach (XmlElement xmlElement2 in list)
			{
				xmlElement.AddNode(xmlElement2);
			}
			return new List<XmlElement> { xmlElement };
		}

		// Token: 0x06000102 RID: 258 RVA: 0x00005FF8 File Offset: 0x000041F8
		private static List<XmlElement> CreateGridContainer(ContainerViewModel containerViewModel, IEnumerable<PrepareAddColumn> fields, PackageKey ProgramKey)
		{
			return UICreator.CreateContainerWithFileds(containerViewModel, fields, ComponentType.Grid, ProgramKey);
		}

		// Token: 0x06000103 RID: 259 RVA: 0x00006004 File Offset: 0x00004204
		private static List<XmlElement> CreateGroupContainer(ContainerViewModel containerViewModel, IEnumerable<PrepareAddColumn> fields, PackageKey ProgramKey)
		{
			List<XmlElement> list = UICreator.CreateGridContainer(containerViewModel, fields, ProgramKey);
			XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(ProgramKey, ComponentType.Group, "");
			xmlElement.AddNode(list.FirstOrDefault<XmlElement>());
			return new List<XmlElement> { xmlElement };
		}

		// Token: 0x06000104 RID: 260 RVA: 0x00006044 File Offset: 0x00004244
		private static List<XmlElement> CreateScrollGridContainer(ContainerViewModel containerViewModel, IEnumerable<PrepareAddColumn> fields, PackageKey ProgramKey)
		{
			XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(ProgramKey, ComponentType.ScrollGrid, null);
			int num = 1;
			int num2 = 12;
			int num3 = 1;
			int num4 = 0;
			foreach (PrepareAddColumn prepareAddColumn in fields)
			{
				string text = UICreator.GetNewNameFromColumnForDetail(ProgramKey, prepareAddColumn);
				if (text.Length != 0)
				{
					XmlElement xmlElement2 = UICreator.CreateFieldComponent(prepareAddColumn, text, ProgramKey);
					xmlElement2.GridX = num2;
					xmlElement2.GridY = num3;
					xmlElement2.SetAttribute("repeat", "true");
					xmlElement2.SetAttribute("rowCount", containerViewModel.RepeatRowCountStrValue);
					xmlElement2.SetAttribute("columnCount", containerViewModel.RepeatColumnCountStrValue);
					xmlElement2.SetAttribute("stepX", "1");
					xmlElement2.SetAttribute("stepY", "0");
					xmlElement2.GridWidth = Math.Min(containerViewModel.MaximumWidthIntValue, prepareAddColumn.WidthIntValue);
					xmlElement2.TabIndex += num4;
					num4++;
					text = ComponentFactory.GetNewName("lbl_" + prepareAddColumn.Column, ProgramKey);
					XmlElement xmlElement3 = ComponentFactory.CreateEmptyComponent(ProgramKey, ComponentType.Label, text);
					xmlElement3.GridWidth = 10;
					xmlElement3.GridX = num;
					xmlElement3.GridY = num3;
					xmlElement3.LocalString = prepareAddColumn.Label;
					xmlElement.AddNode(xmlElement2);
					xmlElement.AddNode(xmlElement3);
					num3 += (containerViewModel.RepeatRowCountIntValue - 1) * xmlElement2.GridHeight + 1;
				}
			}
			return new List<XmlElement> { xmlElement };
		}

		// Token: 0x06000105 RID: 261 RVA: 0x000061EC File Offset: 0x000043EC
		private static List<XmlElement> CreateTableContainer(ContainerViewModel containerViewModel, IEnumerable<PrepareAddColumn> fields, PackageKey ProgramKey)
		{
			XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(ProgramKey, ComponentType.Table, null);
			int num = 0;
			foreach (PrepareAddColumn prepareAddColumn in fields)
			{
				string newNameFromColumnForDetail = UICreator.GetNewNameFromColumnForDetail(ProgramKey, prepareAddColumn);
				if (newNameFromColumnForDetail.Length != 0)
				{
					XmlElement xmlElement2 = UICreator.CreateFieldComponent(prepareAddColumn, newNameFromColumnForDetail, ProgramKey);
					xmlElement2.GridWidth = Math.Min(containerViewModel.MaximumWidthIntValue, prepareAddColumn.WidthIntValue);
					xmlElement2.TabIndex += num;
					num++;
					xmlElement.AddNodeAt(xmlElement2, xmlElement.Nodes.Count);
					xmlElement2.LocalString = prepareAddColumn.Label;
				}
			}
			return new List<XmlElement> { xmlElement };
		}

		// Token: 0x06000106 RID: 262 RVA: 0x000062B8 File Offset: 0x000044B8
		private static List<XmlElement> CreateTreeContainer(ContainerViewModel containerViewModel, IEnumerable<PrepareAddColumn> fields, PackageKey ProgramKey)
		{
			List<XmlElement> list = ComponentFactory.CreateEmptyComponentWithLabel(ProgramKey, ComponentType.Tree);
			int num = 1;
			foreach (PrepareAddColumn prepareAddColumn in fields)
			{
				string newNameFromColumnForDetail = UICreator.GetNewNameFromColumnForDetail(ProgramKey, prepareAddColumn);
				if (newNameFromColumnForDetail.Length != 0)
				{
					XmlElement xmlElement = UICreator.CreateFieldComponent(prepareAddColumn, newNameFromColumnForDetail, ProgramKey);
					xmlElement.GridWidth = Math.Min(containerViewModel.MaximumWidthIntValue, prepareAddColumn.WidthIntValue);
					xmlElement.TabIndex += num;
					num++;
					list[0].AddNode(xmlElement);
					xmlElement.LocalString = prepareAddColumn.Label;
				}
			}
			return list;
		}

		// Token: 0x06000107 RID: 263 RVA: 0x0000636C File Offset: 0x0000456C
		private static SpecificationInfo GetSpecificationInfo(PackageKey key)
		{
			return SettingManager.Get().GetTzpManger(key).SpecificationInfo;
		}

		// Token: 0x06000108 RID: 264 RVA: 0x00006380 File Offset: 0x00004580
		private static string GetCurrentNewName(PackageKey ProgramKey, ComponentType type, PrepareAddColumn column)
		{
			SpecificationInfo specificationInfo = UICreator.GetSpecificationInfo(ProgramKey);
			string text = string.Format("{0}.{1}", column.Table, column.Column);
			string text3;
			string text2;
			bool flag;
			if (type == ComponentType.Label)
			{
				text2 = (text3 = string.Format("lbl_{0}", column.Column));
				flag = specificationInfo.IsExists(text2) || UICreator.Named.Contains(text2);
			}
			else
			{
				text2 = column.Column;
				text3 = text;
				flag = specificationInfo.IsExists(text) || UICreator.Named.Contains(text);
			}
			int num = 0;
			while (flag)
			{
				if (num == 0)
				{
					text3 = text2;
				}
				else
				{
					text3 = string.Format("{0}_{1}", text2, num);
				}
				flag = specificationInfo.IsExists(text3) || UICreator.Named.Contains(text3);
				num++;
			}
			text2 = text3;
			UICreator.Named.Add(text2);
			return text2;
		}

		// Token: 0x04000094 RID: 148
		private static HashSet<string> Named = new HashSet<string>();
	}
}
