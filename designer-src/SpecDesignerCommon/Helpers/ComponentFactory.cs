using System;
using System.Collections.Generic;
using System.Linq;
using System.Text.RegularExpressions;
using System.Windows;
using System.Xml.Linq;
using SpecDesignerCommon.ViewModel;

namespace SpecDesignerCommon.Helpers
{
	// Token: 0x02000108 RID: 264
	public static class ComponentFactory
	{
		// Token: 0x0600092D RID: 2349 RVA: 0x0002C320 File Offset: 0x0002A520
		public static void SetSpecification(string CoreBrString, ModFdInfo modFdInfo)
		{
			if (ComponentFactory._modFdInfo == null)
			{
				ComponentFactory._modFdInfo = modFdInfo;
			}
			if (ComponentFactory._core_br == null)
			{
				ComponentFactory._core_br = XElement.Parse(CoreBrString);
			}
			if (ComponentFactory._nodeInfoList == null)
			{
				ComponentFactory._nodeInfoList = new XElement("NodeInfoList");
			}
			if (ComponentFactory._propertyInfoList == null)
			{
				ComponentFactory._propertyInfoList = new XElement("PropertyInfoList");
			}
			ComponentFactory._propertyInfoList.RemoveNodes();
			ComponentFactory._nodeInfoList.RemoveNodes();
			ComponentFactory._propertyInfoList.Add(ComponentFactory._core_br.Descendants("PropertyInfo"));
			ComponentFactory._nodeInfoList.Add(ComponentFactory._core_br.Descendants("NodeInfo"));
			XElement xelement = XElement.Parse(ComponentFactory._modFdInfo.Content);
			ComponentFactory._propertyInfoList.Add(xelement.Descendants("PropertyInfo"));
			foreach (XElement xelement2 in xelement.Descendants("NodeInfo"))
			{
				if (xelement2.Attribute("properties") != null)
				{
					ComponentFactory._nodeInfoList.Add(xelement2);
				}
			}
		}

		// Token: 0x0600092E RID: 2350 RVA: 0x0002C45C File Offset: 0x0002A65C
		public static List<XmlElement> CreateEmptyComponentWithLabel(PackageKey key, ComponentType componentType)
		{
			List<XmlElement> list = new List<XmlElement>();
			XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(key, componentType, string.Empty);
			XmlElement xmlElement2 = null;
			if (componentType <= ComponentType.DateEdit)
			{
				if (componentType == ComponentType.Folder)
				{
					XmlElement xmlElement3 = ComponentFactory.CreateEmptyComponent(key, ComponentType.Page, string.Empty);
					xmlElement.AddNode(xmlElement3);
					goto IL_0205;
				}
				switch (componentType)
				{
				case ComponentType.Table:
				{
					XmlElement xmlElement4 = ComponentFactory.CreateEmptyComponent(key, ComponentType.Edit, string.Empty);
					xmlElement.AddNode(xmlElement4);
					goto IL_0205;
				}
				case ComponentType.Tree:
				{
					string newName = ComponentFactory.GetNewName("id", key);
					string newName2 = ComponentFactory.GetNewName("parentid", key);
					string newName3 = ComponentFactory.GetNewName("isnode", key);
					string newName4 = ComponentFactory.GetNewName("expanded", key);
					string newName5 = ComponentFactory.GetNewName("name", key);
					XmlElement xmlElement5 = ComponentFactory.CreateEmptyComponent(key, ComponentType.Edit, newName5);
					XmlElement xmlElement6 = ComponentFactory.CreateEmptyComponent(key, ComponentType.Phantom, newName);
					XmlElement xmlElement7 = ComponentFactory.CreateEmptyComponent(key, ComponentType.Phantom, newName2);
					XmlElement xmlElement8 = ComponentFactory.CreateEmptyComponent(key, ComponentType.Phantom, newName3);
					XmlElement xmlElement9 = ComponentFactory.CreateEmptyComponent(key, ComponentType.Phantom, newName4);
					xmlElement.AddNode(xmlElement5);
					xmlElement.AddNodeAt(xmlElement6, xmlElement.Nodes.Count);
					xmlElement.AddNodeAt(xmlElement7, xmlElement.Nodes.Count);
					xmlElement.AddNodeAt(xmlElement8, xmlElement.Nodes.Count);
					xmlElement.AddNodeAt(xmlElement9, xmlElement.Nodes.Count);
					xmlElement.SetAttribute("idColumn", newName);
					xmlElement.SetAttribute("parentIdColumn", newName2);
					xmlElement.SetAttribute("expandedColumn", newName4);
					xmlElement.SetAttribute("isNodeColumn", newName3);
					goto IL_0205;
				}
				case ComponentType.VBox:
				case ComponentType.Label:
				case ComponentType.ProgressBar:
				case ComponentType.Button:
					goto IL_0205;
				case ComponentType.Edit:
				case ComponentType.ComboBox:
				case ComponentType.TextEdit:
				case ComponentType.ButtonEdit:
				case ComponentType.DateEdit:
					break;
				default:
					goto IL_0205;
				}
			}
			else
			{
				switch (componentType)
				{
				case ComponentType.SpinEdit:
				case ComponentType.TimeEdit:
					break;
				default:
					if (componentType != ComponentType.DateTimeEdit)
					{
						goto IL_0205;
					}
					break;
				}
			}
			string newName6 = ComponentFactory.GetNewName(string.Format("{0}_1", xmlElement.GetAttribute("name")), key);
			xmlElement2 = ComponentFactory.CreateEmptyComponent(key, ComponentType.Label, newName6);
			xmlElement2.BindElement = xmlElement;
			xmlElement.BindElement = xmlElement2;
			SettingManager.Get().GetTzpManger(key).SpecificationInfo.AddSpecBinding(xmlElement2, xmlElement);
			IL_0205:
			if (xmlElement2 != null)
			{
				list.Add(xmlElement2);
			}
			list.Add(xmlElement);
			return list;
		}

		// Token: 0x0600092F RID: 2351 RVA: 0x0002C680 File Offset: 0x0002A880
		public static XmlElement CreateEmptyComponentForBody(PackageKey key, SpecNodeType specType)
		{
			XmlElement xmlElement = null;
			switch (specType)
			{
			case SpecNodeType.PROGREL:
			{
				xmlElement = ComponentFactory.CreateEmptyComponent(key, ComponentType.FFLabel, "");
				string newName = ComponentFactory.GetNewName(string.Format("{0}{1}", "prog_", ComponentType.FFLabel.ToString().ToLower()), key);
				xmlElement.SetAttribute("name", newName);
				xmlElement.SetAttribute("tag", "sync");
				xmlElement.SetAttribute("style", "textFormat_html");
				xmlElement.SetAttribute("sizePolicy", "fixed");
				xmlElement.SetAttribute("gridWidth", "10");
				break;
			}
			case SpecNodeType.REFERENCE:
				xmlElement = ComponentFactory.CreateEmptyComponent(key, ComponentType.Edit, "");
				xmlElement.SetAttribute("style", "reference");
				xmlElement.SetAttribute("sizePolicy", "fixed");
				xmlElement.SetAttribute("gridWidth", "10");
				xmlElement.SetAttribute("noEntry", "true");
				break;
			case SpecNodeType.MULTILANG:
				xmlElement = ComponentFactory.CreateEmptyComponent(key, ComponentType.ButtonEdit, "");
				xmlElement.SetAttribute("image", "16/langmodify.png");
				xmlElement.SetAttribute("action", "update_item");
				break;
			}
			return xmlElement;
		}

		// Token: 0x06000930 RID: 2352 RVA: 0x0002C7B4 File Offset: 0x0002A9B4
		public static XmlElement CreateEmptyComponent(PackageKey key, SpecNodeType specType)
		{
			switch (specType)
			{
			case SpecNodeType.PROGREL:
			{
				string newName = ComponentFactory.GetNewName("prog_" + ComponentType.Button.ToString().ToLower(), key);
				XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(key, ComponentType.Button, newName);
				xmlElement.SetAttribute("style", "button_qrystr");
				return xmlElement;
			}
			case SpecNodeType.REFERENCE:
			{
				string newName2 = ComponentFactory.GetNewName(ComponentType.FFLabel.ToString().ToLower() + "_desc", key);
				XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(key, ComponentType.FFLabel, newName2);
				xmlElement.SetAttribute("style", "reference");
				xmlElement.SetAttribute("sizePolicy", "fixed");
				xmlElement.SetAttribute("gridWidth", "10");
				return xmlElement;
			}
			case SpecNodeType.MULTILANG:
			{
				XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(key, ComponentType.ButtonEdit, "");
				xmlElement.SetAttribute("image", "16/langmodify.png");
				xmlElement.SetAttribute("action", "update_item");
				return xmlElement;
			}
			}
			throw new Exception(string.Format(Application.Current.FindResource("specMessage_createFail") as string, specType));
		}

		// Token: 0x06000931 RID: 2353 RVA: 0x0002C8D8 File Offset: 0x0002AAD8
		public static XmlElement CreateEmptyComponent(PackageKey key, ComponentType componentType, string componentName)
		{
			if (string.IsNullOrEmpty(componentName))
			{
				componentName = ComponentFactory.GetNewName(componentType.ToString(), key);
			}
			XmlElement xmlElement = ComponentFactory.CreateEmptyElementWithName(componentType, componentName, key);
			switch (componentType)
			{
			case ComponentType.Page:
				if (xmlElement.GetAttribute("action") == null)
				{
					xmlElement.SetAttribute("action", "");
				}
				break;
			case ComponentType.RadioGroup:
			case ComponentType.ComboBox:
				if (xmlElement.GetAttribute("items") == null || XmlElement.NOSET == xmlElement.GetAttribute("items"))
				{
					xmlElement.SetAttribute("items", "");
				}
				if (xmlElement.GetAttribute("case") == null || XmlElement.NOSET == xmlElement.GetAttribute("case"))
				{
					xmlElement.SetAttribute("case", "none");
				}
				if (xmlElement.GetAttribute("noEntry") == null || XmlElement.NOSET == xmlElement.GetAttribute("noEntry"))
				{
					xmlElement.SetAttribute("noEntry", "false");
				}
				break;
			case ComponentType.Table:
				if (xmlElement.GetAttribute("unsortableColumns") == null)
				{
					xmlElement.SetAttribute("unsortableColumns", "");
				}
				if (xmlElement.GetAttribute("unsizableColumns") == null)
				{
					xmlElement.SetAttribute("unsizableColumns", "");
				}
				if (xmlElement.GetAttribute("unmovableColumns") == null)
				{
					xmlElement.SetAttribute("unmovableColumns", "");
				}
				if (xmlElement.GetAttribute("unhidableColumns") == null)
				{
					xmlElement.SetAttribute("unhidableColumns", "");
				}
				if (xmlElement.GetAttribute("text") != null)
				{
					xmlElement.RemoveAttribute("text");
				}
				if (xmlElement.GetAttribute("totalRows") == "")
				{
					xmlElement.SetAttribute("totalRows", "5");
				}
				if (xmlElement.GetAttribute("width") == "0" && xmlElement.GetAttribute("unitWidth") == "characters")
				{
					xmlElement.SetAttribute("width", XmlElement.NOSET);
					xmlElement.SetAttribute("unitWidth", XmlElement.NOSET);
				}
				if (xmlElement.GetAttribute("height") == "0" && xmlElement.GetAttribute("unitHeight") == "characters")
				{
					xmlElement.SetAttribute("height", XmlElement.NOSET);
					xmlElement.SetAttribute("unitHeight", XmlElement.NOSET);
				}
				break;
			case ComponentType.Tree:
				if (xmlElement.GetAttribute("unsortableColumns") == null)
				{
					xmlElement.SetAttribute("unsortableColumns", "");
				}
				if (xmlElement.GetAttribute("unsizableColumns") == null)
				{
					xmlElement.SetAttribute("unsizableColumns", "");
				}
				if (xmlElement.GetAttribute("unmovableColumns") == null)
				{
					xmlElement.SetAttribute("unmovableColumns", "");
				}
				if (xmlElement.GetAttribute("unhidableColumns") == null)
				{
					xmlElement.SetAttribute("unhidableColumns", "");
				}
				if (xmlElement.GetAttribute("text") != null)
				{
					xmlElement.RemoveAttribute("text");
				}
				if (xmlElement.GetAttribute("totalRows") == "")
				{
					xmlElement.SetAttribute("totalRows", "5");
				}
				if (xmlElement.GetAttribute("width") == "0" && xmlElement.GetAttribute("unitWidth") == "characters")
				{
					xmlElement.SetAttribute("width", XmlElement.NOSET);
					xmlElement.SetAttribute("unitWidth", XmlElement.NOSET);
				}
				if (xmlElement.GetAttribute("height") == "0" && xmlElement.GetAttribute("unitHeight") == "characters")
				{
					xmlElement.SetAttribute("height", XmlElement.NOSET);
					xmlElement.SetAttribute("unitHeight", XmlElement.NOSET);
				}
				break;
			case ComponentType.VBox:
				if (xmlElement.GetAttribute("splitter") == null || XmlElement.NOSET == xmlElement.GetAttribute("splitter"))
				{
					xmlElement.SetAttribute("splitter", "false");
				}
				break;
			case ComponentType.Edit:
				if (xmlElement.GetAttribute("invisible") == null || XmlElement.NOSET == xmlElement.GetAttribute("invisible"))
				{
					xmlElement.SetAttribute("invisible", "");
				}
				if (xmlElement.GetAttribute("case") == null || XmlElement.NOSET == xmlElement.GetAttribute("case"))
				{
					xmlElement.SetAttribute("case", "none");
				}
				if (xmlElement.GetAttribute("noEntry") == null || XmlElement.NOSET == xmlElement.GetAttribute("noEntry"))
				{
					xmlElement.SetAttribute("noEntry", "false");
				}
				break;
			case ComponentType.TextEdit:
				if (xmlElement.GetAttribute("case") == null || XmlElement.NOSET == xmlElement.GetAttribute("case"))
				{
					xmlElement.SetAttribute("case", "none");
				}
				if (xmlElement.GetAttribute("noEntry") == null || XmlElement.NOSET == xmlElement.GetAttribute("noEntry"))
				{
					xmlElement.SetAttribute("noEntry", "false");
				}
				break;
			case ComponentType.ButtonEdit:
				if (xmlElement.GetAttribute("invisible") == null || XmlElement.NOSET == xmlElement.GetAttribute("invisible"))
				{
					xmlElement.SetAttribute("invisible", "");
				}
				if (xmlElement.GetAttribute("case") == null || XmlElement.NOSET == xmlElement.GetAttribute("case"))
				{
					xmlElement.SetAttribute("case", "none");
				}
				if (xmlElement.GetAttribute("noEntry") == null || XmlElement.NOSET == xmlElement.GetAttribute("noEntry"))
				{
					xmlElement.SetAttribute("noEntry", "false");
				}
				if (xmlElement.GetAttribute("completer") == null || XmlElement.NOSET == xmlElement.GetAttribute("completer"))
				{
					xmlElement.SetAttribute("completer", "false");
				}
				break;
			case ComponentType.CheckBox:
				if (xmlElement.GetAttribute("valueChecked") == null)
				{
					xmlElement.SetAttribute("valueChecked", "Y");
				}
				if (xmlElement.GetAttribute("valueUnchecked") == null)
				{
					xmlElement.SetAttribute("valueUnchecked", "N");
				}
				break;
			case ComponentType.SpinEdit:
				if (xmlElement.GetAttribute("invisible") == null || XmlElement.NOSET == xmlElement.GetAttribute("invisible"))
				{
					xmlElement.SetAttribute("invisible", "");
				}
				break;
			case ComponentType.TimeEdit:
				if (xmlElement.GetAttribute("invisible") == null || XmlElement.NOSET == xmlElement.GetAttribute("invisible"))
				{
					xmlElement.SetAttribute("invisible", "");
				}
				break;
			}
			if (xmlElement.GetAttribute("columnCount") != null)
			{
				xmlElement.SetAttribute("columnCount", "2");
			}
			if (xmlElement.GetAttribute("rowCount") != null)
			{
				xmlElement.SetAttribute("rowCount", "1");
			}
			if (xmlElement.GetAttribute("stepX") != null)
			{
				xmlElement.SetAttribute("stepX", "1");
			}
			if (xmlElement.GetAttribute("stepY") != null)
			{
				xmlElement.SetAttribute("stepY", "0");
			}
			if (xmlElement.GetAttribute("posX") != null && xmlElement.GetAttribute("posX") == string.Empty)
			{
				xmlElement.SetAttribute("posX", "0");
			}
			if (xmlElement.GetAttribute("posY") != null && xmlElement.GetAttribute("posY") == string.Empty)
			{
				xmlElement.SetAttribute("posY", "0");
			}
			if (xmlElement.GetAttribute("gridWidth") != null && xmlElement.GetAttribute("gridWidth") != XmlElement.NOSET)
			{
				int num = (int)short.Parse(xmlElement.GetAttribute("gridWidth"));
				if (FormDesignSetting.IsContainer(xmlElement.NodeName))
				{
					num = 19;
				}
				if (ComponentType.TextEdit == xmlElement.Type)
				{
					num = 20;
				}
				xmlElement.SetAttribute("gridWidth", num.ToString());
			}
			if (xmlElement.GetAttribute("gridHeight") != null && xmlElement.GetAttribute("gridHeight") != XmlElement.NOSET)
			{
				int num2 = (int)short.Parse(xmlElement.GetAttribute("gridHeight"));
				if (FormDesignSetting.IsContainer(xmlElement.NodeName))
				{
					num2 = 6;
				}
				if (ComponentType.TextEdit == xmlElement.Type)
				{
					num2 = 3;
				}
				xmlElement.SetAttribute("gridHeight", num2.ToString());
			}
			if (ComponentFactory.IsIncludeProperties(xmlElement.NodeName, "tabIndex"))
			{
				SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(key).SpecificationInfo;
				int num3 = 0;
				XmlElement.GetMaxTabIndex(specificationInfo.FormNode, ref num3);
				num3++;
				xmlElement.SetAttribute("tabIndex", num3.ToString());
			}
			return xmlElement;
		}

		// Token: 0x06000932 RID: 2354 RVA: 0x0002D1B4 File Offset: 0x0002B3B4
		public static string GetNewName(string defaultName, PackageKey key)
		{
			string text = "_";
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(key).SpecificationInfo;
			ComponentType componentType;
			if (ComponentType.Table.ToString().Equals(defaultName, StringComparison.CurrentCultureIgnoreCase) || ComponentType.Tree.ToString().Equals(defaultName, StringComparison.CurrentCultureIgnoreCase) || ComponentType.ScrollGrid.ToString().Equals(defaultName, StringComparison.CurrentCultureIgnoreCase))
			{
				defaultName = "s_detail";
				text = "";
				string text2 = string.Format("{0}1", defaultName);
				if (!specificationInfo.IsExists(text2))
				{
					return text2;
				}
			}
			else if (!string.IsNullOrWhiteSpace(defaultName) && !specificationInfo.IsExists(defaultName) && !Enum.TryParse<ComponentType>(defaultName, true, out componentType))
			{
				return defaultName;
			}
			ComponentType componentType2;
			if (Enum.TryParse<ComponentType>(defaultName, true, out componentType2))
			{
				string text3 = string.Format("{0}{1}1", componentType2.ToString().ToLower(), text);
				if (!specificationInfo.IsExists(text3))
				{
					return text3;
				}
			}
			List<int> list = new List<int>();
			foreach (KeyValuePair<string, FormSpecModel> keyValuePair in specificationInfo.FormSpeDictionary)
			{
				if (keyValuePair.Key.StartsWith(defaultName, StringComparison.CurrentCultureIgnoreCase))
				{
					list.Add(ComponentFactory.GetSequenceNumber(keyValuePair.Key, defaultName));
				}
			}
			if (defaultName == "s_detail" && (!SettingManager.Get().GetTzpManger(key).IsStandardProgram || specificationInfo.Env == "c"))
			{
				int num = 40;
				if (specificationInfo.Env == "c")
				{
					num = 70;
				}
				int num2;
				if (list.Max() <= num)
				{
					num2 = num + 1;
				}
				else
				{
					num2 = list.Max() + 1;
				}
				return defaultName + text + num2.ToString();
			}
			if (list.Count != 0)
			{
				return string.Format("{0}{1}{2}", defaultName, text, (list.Count == 0) ? "1" : (list.Max() + 1).ToString());
			}
			return defaultName;
		}

		// Token: 0x06000933 RID: 2355 RVA: 0x0002D3B4 File Offset: 0x0002B5B4
		private static int GetSequenceNumber(string name, string namePrefix)
		{
			string text = "(?=" + namePrefix.ToLowerInvariant() + "\\D{0,}(\\d*$))";
			Match match = Regex.Match(name.ToLowerInvariant(), text);
			if (match.Success && !"".Equals(match.Groups[1].Value))
			{
				return (int)short.Parse(match.Groups[1].Value);
			}
			return 0;
		}

		// Token: 0x06000934 RID: 2356 RVA: 0x0002D424 File Offset: 0x0002B624
		internal static XmlElement CreateEmptyElementWithName(ComponentType componentType, string componentName, PackageKey key)
		{
			SpecificationInfo specificationInfo = SettingManager.Get().GetTzpManger(key).SpecificationInfo;
			XmlElement xmlElement = new XmlElement(key, componentType);
			if (xmlElement != null)
			{
				if (componentType == ComponentType.RadioGroupItem || componentType == ComponentType.Item)
				{
					xmlElement.SetAttribute("name", componentName);
				}
				else
				{
					xmlElement.SetAttribute("name", componentName.ToLower());
				}
				ComponentFactory.AttachDefaultAttributes(key, xmlElement);
				if (xmlElement.GetAttribute("text") != null && !string.IsNullOrEmpty(xmlElement.Name))
				{
					switch (componentType)
					{
					case ComponentType.Group:
					case ComponentType.Page:
						if (xmlElement.GetAttribute("text") == string.Empty)
						{
							xmlElement.SetAttribute("text", string.Format("{0}{1}{2}", key.Program, ".", xmlElement.Name));
						}
						if (xmlElement.GetAttribute("comment") == string.Empty)
						{
							xmlElement.SetAttribute("comment", string.Format("{0}{1}cmt_{2}", key.Program, ".", xmlElement.Name));
						}
						break;
					case ComponentType.HBox:
						break;
					default:
						if ((componentType == ComponentType.Label || componentType == ComponentType.Button) && xmlElement.GetAttribute("text") == string.Empty)
						{
							xmlElement.SetAttribute("text", xmlElement.Name);
						}
						break;
					}
				}
				if ((xmlElement.GetAttribute("title") != null && xmlElement.GetAttribute("title") == string.Empty) || xmlElement.GetAttribute("title") == XmlElement.NOSET)
				{
					xmlElement.SetAttribute("title", xmlElement.Name);
				}
				if (xmlElement.Type != ComponentType.Phantom && xmlElement.Type != ComponentType.Item && xmlElement.GetAttribute("tag") == null)
				{
					xmlElement.SetAttribute("tag", "");
				}
				if (xmlElement.GetAttribute("comment") != null && xmlElement.GetAttribute("comment") == "")
				{
					if (componentName.StartsWith("lbl_"))
					{
						xmlElement.SetAttribute("comment", string.Format("cmt_{0}", componentName.Substring(4).ToLower()));
					}
					else
					{
						xmlElement.SetAttribute("comment", string.Format("cmt_{0}", componentName.ToLower()));
					}
				}
				if (!FormDesignSetting.IsContainer(xmlElement.NodeName) && componentType != ComponentType.Button)
				{
					string empty = string.Empty;
					if (xmlElement.GetAttribute("title") != null)
					{
						xmlElement.SetAttribute("title", string.Format("lbl_{0}", xmlElement.GetAttribute("name")));
					}
					if (xmlElement.GetAttribute("text") != null && componentName.StartsWith("lbl_"))
					{
						xmlElement.SetAttribute("text", string.Format("{0}", xmlElement.GetAttribute("name")));
					}
				}
			}
			return xmlElement;
		}

		// Token: 0x06000935 RID: 2357 RVA: 0x0002D6D9 File Offset: 0x0002B8D9
		internal static XmlElement AttachDefaultAttributes(PackageKey key, XmlElement nodeElement)
		{
			return ComponentFactory.AttachDefaultAttributes(key, nodeElement, true);
		}

		// Token: 0x06000936 RID: 2358 RVA: 0x0002D724 File Offset: 0x0002B924
		internal static XmlElement AttachDefaultAttributes(PackageKey key, XmlElement nodeElement, bool needDefaultValue)
		{
			ComponentType componentType = nodeElement.Type;
			if (componentType == ComponentType.Unknown)
			{
				componentType = ComponentType.Edit;
			}
			string attribute = nodeElement.GetAttribute("name");
			if (ComponentFactory._propertyInfoList == null || ComponentFactory._nodeInfoList == null)
			{
				MessageBox.Show("can't find core-br.spec and mod-fd.spec files!");
			}
			XElement xelement = (from comp in ComponentFactory._nodeInfoList.Descendants("NodeInfo")
				where ("modFD/" + componentType.ToString()).Equals(comp.Attribute("mimeType").Value)
				select comp).SingleOrDefault<XElement>();
			if (xelement == null && componentType.ToString() == "DateTimeEdit" && SettingManager.Get().ErpVer == "1.0")
			{
				return nodeElement;
			}
			if (xelement == null)
			{
				throw new NullReferenceException(componentType.ToString() + " not found!");
			}
			XAttribute xattribute = xelement.Attribute("properties");
			if (xattribute == null)
			{
				throw new NullReferenceException("properties attribute not found!");
			}
			string[] array = xattribute.Value.Split(new char[] { ';' });
			ComponentType componentType2 = componentType;
			if (componentType2 <= ComponentType.Tree)
			{
				if (componentType2 != ComponentType.Folder)
				{
					switch (componentType2)
					{
					case ComponentType.HBox:
						if (nodeElement.GetAttribute("splitter") == null)
						{
							nodeElement.SetAttribute("splitter", (!needDefaultValue) ? XmlElement.NOSET : "false");
						}
						break;
					case ComponentType.RadioGroup:
						if (nodeElement.GetAttribute("gridWidth") == null)
						{
							nodeElement.SetAttribute("gridWidth", (!needDefaultValue) ? XmlElement.NOSET : "20");
						}
						if (nodeElement.GetAttribute("gridHeight") == null)
						{
							nodeElement.SetAttribute("gridHeight", (!needDefaultValue) ? XmlElement.NOSET : "1");
						}
						break;
					case ComponentType.Table:
					case ComponentType.Tree:
						if (componentType == ComponentType.Table && nodeElement.GetAttribute("doubleClick") == null)
						{
							nodeElement.SetAttribute("doubleClick", (!needDefaultValue) ? XmlElement.NOSET : "modify_detail");
						}
						if (nodeElement.GetAttribute("totalRows") == null)
						{
							nodeElement.SetAttribute("totalRows", "5");
						}
						if (nodeElement.GetAttribute("width") == "0" && nodeElement.GetAttribute("unitWidth") == "characters")
						{
							nodeElement.SetAttribute("width", XmlElement.NOSET);
							nodeElement.SetAttribute("unitWidth", XmlElement.NOSET);
						}
						if (nodeElement.GetAttribute("height") == "0" && nodeElement.GetAttribute("unitHeight") == "characters")
						{
							nodeElement.SetAttribute("height", XmlElement.NOSET);
							nodeElement.SetAttribute("unitHeight", XmlElement.NOSET);
						}
						break;
					}
				}
				else if (nodeElement.GetAttribute("style") == null)
				{
					nodeElement.SetAttribute("style", XmlElement.NOSET);
				}
			}
			else
			{
				switch (componentType2)
				{
				case ComponentType.ComboBox:
					if (nodeElement.GetAttribute("scroll") == null)
					{
						nodeElement.SetAttribute("scroll", (!needDefaultValue) ? XmlElement.NOSET : "true");
					}
					if (nodeElement.GetAttribute("gridWidth") == null)
					{
						nodeElement.SetAttribute("gridWidth", (!needDefaultValue) ? XmlElement.NOSET : "10");
					}
					if (nodeElement.GetAttribute("gridHeight") == null)
					{
						nodeElement.SetAttribute("gridHeight", (!needDefaultValue) ? XmlElement.NOSET : "1");
					}
					if (nodeElement.GetAttribute("queryEditable") == null)
					{
						nodeElement.SetAttribute("queryEditable", (!needDefaultValue) ? XmlElement.NOSET : "true");
					}
					break;
				case ComponentType.TextEdit:
				case ComponentType.Button:
				case ComponentType.DateEdit:
				case ComponentType.RadioGroupItem:
				case ComponentType.Canvas:
				case ComponentType.FFLabel:
					break;
				case ComponentType.ButtonEdit:
					if (nodeElement.GetAttribute("image") == null)
					{
						nodeElement.SetAttribute("image", "16/openwindow.png");
					}
					if (nodeElement.GetAttribute("action") == null)
					{
						nodeElement.SetAttribute("action", "controlp");
					}
					break;
				case ComponentType.CheckBox:
					if (nodeElement.GetAttribute("gridWidth") == null)
					{
						nodeElement.SetAttribute("gridWidth", (!needDefaultValue) ? XmlElement.NOSET : "20");
					}
					if (nodeElement.GetAttribute("valueChecked") == null)
					{
						nodeElement.SetAttribute("valueChecked", (!needDefaultValue) ? XmlElement.NOSET : "Y");
					}
					if (nodeElement.GetAttribute("valueUnchecked") == null)
					{
						nodeElement.SetAttribute("valueUnchecked", (!needDefaultValue) ? XmlElement.NOSET : "N");
					}
					break;
				case ComponentType.FFImage:
					if (nodeElement.GetAttribute("justify") == null)
					{
						nodeElement.SetAttribute("justify", (!needDefaultValue) ? XmlElement.NOSET : "center");
					}
					if (nodeElement.GetAttribute("sizePolicy") == null)
					{
						nodeElement.SetAttribute("sizePolicy", (!needDefaultValue) ? XmlElement.NOSET : "dynamic");
					}
					if (nodeElement.GetAttribute("gridWidth") == null)
					{
						nodeElement.SetAttribute("gridWidth", (!needDefaultValue) ? XmlElement.NOSET : "17");
					}
					if (nodeElement.GetAttribute("gridHeight") == null)
					{
						nodeElement.SetAttribute("gridHeight", (!needDefaultValue) ? XmlElement.NOSET : "4");
					}
					if (nodeElement.GetAttribute("width") == null)
					{
						nodeElement.SetAttribute("width", (!needDefaultValue) ? XmlElement.NOSET : "200");
					}
					if (nodeElement.GetAttribute("unitWidth") == null)
					{
						nodeElement.SetAttribute("unitWidth", (!needDefaultValue) ? XmlElement.NOSET : "pixels");
					}
					if (nodeElement.GetAttribute("height") == null)
					{
						nodeElement.SetAttribute("height", (!needDefaultValue) ? XmlElement.NOSET : "150");
					}
					if (nodeElement.GetAttribute("unitHeight") == null)
					{
						nodeElement.SetAttribute("unitHeight", (!needDefaultValue) ? XmlElement.NOSET : "pixels");
					}
					if (nodeElement.GetAttribute("autoScale") == null)
					{
						nodeElement.SetAttribute("autoScale", (!needDefaultValue) ? XmlElement.NOSET : "true");
					}
					if (nodeElement.GetAttribute("stretch") == null)
					{
						nodeElement.SetAttribute("stretch", (!needDefaultValue) ? XmlElement.NOSET : "none");
					}
					if (nodeElement.GetAttribute("style") == null)
					{
						nodeElement.SetAttribute("style", (!needDefaultValue) ? XmlElement.NOSET : "noBorder");
					}
					break;
				case ComponentType.Image:
					if (nodeElement.GetAttribute("sizePolicy") == null)
					{
						nodeElement.SetAttribute("sizePolicy", (!needDefaultValue) ? XmlElement.NOSET : "dynamic");
					}
					if (nodeElement.GetAttribute("width") == null)
					{
						nodeElement.SetAttribute("width", (!needDefaultValue) ? XmlElement.NOSET : "200");
					}
					if (nodeElement.GetAttribute("height") == null)
					{
						nodeElement.SetAttribute("height", (!needDefaultValue) ? XmlElement.NOSET : "150");
					}
					if (nodeElement.GetAttribute("gridWidth") == null)
					{
						nodeElement.SetAttribute("gridWidth", (!needDefaultValue) ? XmlElement.NOSET : "17");
					}
					if (nodeElement.GetAttribute("gridHeight") == null)
					{
						nodeElement.SetAttribute("gridHeight", (!needDefaultValue) ? XmlElement.NOSET : "4");
					}
					if (nodeElement.GetAttribute("unitWidth") == null)
					{
						nodeElement.SetAttribute("unitWidth", (!needDefaultValue) ? XmlElement.NOSET : "pixels");
					}
					if (nodeElement.GetAttribute("unitHeight") == null)
					{
						nodeElement.SetAttribute("unitHeight", (!needDefaultValue) ? XmlElement.NOSET : "pixels");
					}
					if (nodeElement.GetAttribute("autoScale") == null)
					{
						nodeElement.SetAttribute("autoScale", (!needDefaultValue) ? XmlElement.NOSET : "true");
					}
					if (nodeElement.GetAttribute("stretch") == null)
					{
						nodeElement.SetAttribute("stretch", (!needDefaultValue) ? XmlElement.NOSET : "none");
					}
					if (nodeElement.GetAttribute("style") == null)
					{
						nodeElement.SetAttribute("style", (!needDefaultValue) ? XmlElement.NOSET : "noBorder");
					}
					break;
				case ComponentType.Slider:
					if (nodeElement.GetAttribute("step") == null)
					{
						nodeElement.SetAttribute("step", (!needDefaultValue) ? XmlElement.NOSET : "1");
					}
					if (nodeElement.GetAttribute("gridWidth") == null)
					{
						nodeElement.SetAttribute("gridWidth", (!needDefaultValue) ? XmlElement.NOSET : "10");
					}
					if (nodeElement.GetAttribute("gridHeight") == null)
					{
						nodeElement.SetAttribute("gridHeight", (!needDefaultValue) ? XmlElement.NOSET : "1");
					}
					if (nodeElement.GetAttribute("valueMin") == null)
					{
						nodeElement.SetAttribute("valueMin", (!needDefaultValue) ? XmlElement.NOSET : "0");
					}
					if (nodeElement.GetAttribute("valueMax") == null)
					{
						nodeElement.SetAttribute("valueMax", (!needDefaultValue) ? XmlElement.NOSET : "5");
					}
					break;
				case ComponentType.SpinEdit:
					if (nodeElement.GetAttribute("step") == null)
					{
						nodeElement.SetAttribute("step", (!needDefaultValue) ? XmlElement.NOSET : "1");
					}
					if (nodeElement.GetAttribute("scroll") == null)
					{
						nodeElement.SetAttribute("scroll", (!needDefaultValue) ? XmlElement.NOSET : "true");
					}
					if (nodeElement.GetAttribute("gridWidth") == null)
					{
						nodeElement.SetAttribute("gridWidth", (!needDefaultValue) ? XmlElement.NOSET : "10");
					}
					if (nodeElement.GetAttribute("gridHeight") == null)
					{
						nodeElement.SetAttribute("gridHeight", (!needDefaultValue) ? XmlElement.NOSET : "1");
					}
					if (nodeElement.GetAttribute("valueMin") == null)
					{
						nodeElement.SetAttribute("valueMin", (!needDefaultValue) ? XmlElement.NOSET : "0");
					}
					if (nodeElement.GetAttribute("valueMax") == null)
					{
						nodeElement.SetAttribute("valueMax", (!needDefaultValue) ? XmlElement.NOSET : "100");
					}
					break;
				case ComponentType.TimeEdit:
					if (nodeElement.GetAttribute("scroll") == null)
					{
						nodeElement.SetAttribute("scroll", (!needDefaultValue) ? XmlElement.NOSET : "true");
					}
					break;
				default:
					if (componentType2 == ComponentType.DateTimeEdit)
					{
						if (nodeElement.GetAttribute("justify") == null)
						{
							nodeElement.SetAttribute("justify", (!needDefaultValue) ? XmlElement.NOSET : "left");
						}
						if (nodeElement.GetAttribute("style") == null)
						{
							nodeElement.SetAttribute("style", (!needDefaultValue) ? XmlElement.NOSET : "dtcanbequery");
						}
						if (nodeElement.GetAttribute("gridWidth") == null)
						{
							nodeElement.SetAttribute("gridWidth", (!needDefaultValue) ? XmlElement.NOSET : "22");
						}
					}
					break;
				}
			}
			foreach (string text in array)
			{
				if (!string.IsNullOrEmpty(text) && nodeElement.GetAttribute(text) == null)
				{
					string text2 = ComponentFactory.GetDefaultAttributeValue(text);
					string text3;
					switch (text3 = text)
					{
					case "displayTabName":
					case "displayColName":
					case "validateTabName":
					case "validateColName":
					case "sample":
					case "sqlType":
					case "lookup":
					case "ref_usage":
					case "guid":
						goto IL_0C84;
					case "scroll":
						text2 = "true";
						break;
					case "justify":
					case "format":
					case "picture":
					case "gridChildrenInParent":
						text2 = XmlElement.NOSET;
						break;
					case "widget":
						text2 = componentType.ToString();
						break;
					case "fontPitch":
						text2 = "";
						break;
					case "gridWidth":
						text2 = "10";
						break;
					case "gridHeight":
						text2 = "1";
						break;
					}
					if ((!text.Contains("aggregate") || ((nodeElement.Parent == null || !(nodeElement.Parent.NodeName != ComponentType.Table.ToString())) && nodeElement.Parent != null)) && text2 != null)
					{
						nodeElement.SetAttribute(text, needDefaultValue ? text2 : ((text2 == "") ? XmlElement.NOSET : text2));
					}
				}
				IL_0C84:;
			}
			if (nodeElement.GetAttribute("name") != null)
			{
				nodeElement.SetAttribute("name", attribute);
			}
			ComponentFactory.RemoveIllegalProperties(nodeElement);
			return nodeElement;
		}

		// Token: 0x06000937 RID: 2359 RVA: 0x0002E3E8 File Offset: 0x0002C5E8
		public static void RemoveIllegalProperties(XmlElement nodeElement)
		{
			ComponentType type = nodeElement.Type;
			if (type != ComponentType.Table)
			{
				return;
			}
			nodeElement.RemoveAttribute("text");
		}

		// Token: 0x06000938 RID: 2360 RVA: 0x0002E410 File Offset: 0x0002C610
		public static void SetPropertiesBaseParent(XmlElement parent, XmlElement nodeElement)
		{
			if (parent == null)
			{
				return;
			}
			switch (parent.Type)
			{
			case ComponentType.ScrollGrid:
				if (nodeElement.GetAttribute("colName") != null)
				{
					if (nodeElement.GetAttribute("stepX") == null)
					{
						nodeElement.SetAttribute("stepX", "1");
					}
					if (nodeElement.GetAttribute("stepY") == null)
					{
						nodeElement.SetAttribute("stepY", "0");
					}
					if (nodeElement.GetAttribute("repeat") == null)
					{
						nodeElement.SetAttribute("repeat", "false");
					}
					if (nodeElement.GetAttribute("rowCount") == null)
					{
						nodeElement.SetAttribute("rowCount", "1");
					}
					if (nodeElement.GetAttribute("columnCount") == null)
					{
						nodeElement.SetAttribute("columnCount", "2");
					}
				}
				if ("0" == nodeElement.GetAttribute("posX"))
				{
					nodeElement.SetAttribute("posX", "1");
				}
				if ("0" == nodeElement.GetAttribute("posY"))
				{
					nodeElement.SetAttribute("posY", "1");
					return;
				}
				break;
			case ComponentType.Table:
				if (nodeElement.Type != ComponentType.Phantom)
				{
					if (nodeElement.GetAttribute("unsortable") == null)
					{
						nodeElement.SetAttribute("unsortable", "false");
					}
					if (nodeElement.GetAttribute("unmovable") == null)
					{
						nodeElement.SetAttribute("unmovable", "false");
					}
					if (nodeElement.GetAttribute("unhidable") == null)
					{
						nodeElement.SetAttribute("unhidable", "false");
					}
					if (nodeElement.GetAttribute("unsizable") == null)
					{
						nodeElement.SetAttribute("unsizable", "false");
					}
					if (nodeElement.GetAttribute("unsizable") == null)
					{
						nodeElement.SetAttribute("unsizable", "false");
					}
					if (nodeElement.GetAttribute("aggregate") == null)
					{
						nodeElement.SetAttribute("aggregate", "false");
						return;
					}
				}
				else
				{
					if (nodeElement.GetAttribute("unsortable") != null)
					{
						nodeElement.RemoveAttribute("unsortable");
					}
					if (nodeElement.GetAttribute("unmovable") != null)
					{
						nodeElement.RemoveAttribute("unmovable");
					}
					if (nodeElement.GetAttribute("unhidable") != null)
					{
						nodeElement.RemoveAttribute("unhidable");
					}
					if (nodeElement.GetAttribute("unsizable") != null)
					{
						nodeElement.RemoveAttribute("unsizable");
					}
				}
				break;
			default:
				return;
			}
		}

		// Token: 0x06000939 RID: 2361 RVA: 0x0002E688 File Offset: 0x0002C888
		private static string GetDefaultAttributeValue(string p)
		{
			XElement xelement = (from prop in ComponentFactory._propertyInfoList.Descendants()
				where prop.Attribute("name") != null && p.Equals(prop.Attribute("name").Value)
				select prop).SingleOrDefault<XElement>();
			if (xelement == null)
			{
				return "";
			}
			if (xelement.Attribute("initialValue") == null)
			{
				return "";
			}
			string value = xelement.Attribute("initialValue").Value;
			if (value != null)
			{
				return value;
			}
			return "";
		}

		// Token: 0x0600093A RID: 2362 RVA: 0x0002E704 File Offset: 0x0002C904
		public static bool IsIncludeProperties(string type, string attr)
		{
			return ComponentFactory._modFdInfo != null && ComponentFactory._modFdInfo.IsIncludeAttribute(type, attr);
		}

		// Token: 0x0600093B RID: 2363 RVA: 0x0002E770 File Offset: 0x0002C970
		public static XElement CreateRecord()
		{
			ComponentFactory.<>c__DisplayClass8 CS$<>8__locals1 = new ComponentFactory.<>c__DisplayClass8();
			CS$<>8__locals1.name = "BR/Record";
			XElement xelement = (from r in ComponentFactory._core_br.Descendants("NodeInfo")
				where CS$<>8__locals1.name.Equals(r.Attribute("mimeType").Value)
				select r).First<XElement>();
			XElement xelement2 = new XElement("Record");
			string[] array = xelement.Attribute("properties").Value.Split(new char[] { ';' });
			XElement xelement3 = ComponentFactory._core_br.Element("PropertyInfoList");
			foreach (string property in array)
			{
				if (!string.IsNullOrEmpty(property) && !(property == "uid"))
				{
					XElement xelement4 = (from p in xelement3.Elements()
						where property.Equals(p.Attribute("name").Value)
						select p).First<XElement>();
					XAttribute xattribute = xelement4.Attribute("initialValue");
					string text = ((xattribute == null) ? "" : xattribute.Value);
					xelement2.Add(new XAttribute(property, text));
				}
			}
			return xelement2;
		}

		// Token: 0x0600093C RID: 2364 RVA: 0x0002E924 File Offset: 0x0002CB24
		public static XElement CreateRecordField()
		{
			ComponentFactory.<>c__DisplayClassf CS$<>8__locals1 = new ComponentFactory.<>c__DisplayClassf();
			CS$<>8__locals1.name = "BR/RecordField";
			XElement xelement = (from r in ComponentFactory._core_br.Descendants("NodeInfo")
				where CS$<>8__locals1.name.Equals(r.Attribute("mimeType").Value)
				select r).First<XElement>();
			XElement xelement2 = new XElement("RecordField");
			string[] array = xelement.Attribute("properties").Value.Split(new char[] { ';' });
			XElement xelement3 = ComponentFactory._core_br.Element("PropertyInfoList");
			foreach (string property in array)
			{
				if (!string.IsNullOrEmpty(property) && !property.Equals("lookup"))
				{
					XElement xelement4 = (from p in xelement3.Elements()
						where property.Equals(p.Attribute("name").Value)
						select p).First<XElement>();
					if (xelement4.Attribute("isHidden") == null && xelement4.Attribute("isPhantom") == null)
					{
						XAttribute xattribute = xelement4.Attribute("initialValue");
						string text = ((xattribute == null) ? "" : xattribute.Value);
						xelement2.Add(new XAttribute(property, text));
					}
				}
			}
			return xelement2;
		}

		// Token: 0x0600093D RID: 2365 RVA: 0x0002EAB0 File Offset: 0x0002CCB0
		public static bool AcceptMimes(ComponentType parentType, XmlElement child)
		{
			bool flag = ComponentFactory.AcceptMimes(parentType.ToString(), child.Type.ToString());
			if (flag)
			{
				switch (parentType)
				{
				case ComponentType.Grid:
				case ComponentType.Group:
					if (FormDesignSetting.IsContainer(child.Type.ToString()))
					{
						flag = false;
					}
					break;
				}
			}
			return flag;
		}

		// Token: 0x0600093E RID: 2366 RVA: 0x0002EB10 File Offset: 0x0002CD10
		public static bool AcceptMimes(XmlElement parent, XmlElement child)
		{
			return ComponentFactory.AcceptMimes(parent, child.Type);
		}

		// Token: 0x0600093F RID: 2367 RVA: 0x0002EB2C File Offset: 0x0002CD2C
		public static bool AcceptMimes(XmlElement parent, ComponentType childType)
		{
			bool flag = ComponentFactory.AcceptMimes(parent.Type.ToString(), childType.ToString());
			if (flag)
			{
				switch (parent.Type)
				{
				case ComponentType.Grid:
					break;
				case ComponentType.Group:
				{
					flag = parent.Parent == null || parent.Parent.Type != ComponentType.Grid || !FormDesignSetting.IsContainer(childType.ToString());
					flag &= childType != ComponentType.Group;
					if (!flag || parent.Parent == null || parent.Parent.Nodes.Count <= 1)
					{
						return flag;
					}
					using (IEnumerator<XmlElement> enumerator = parent.Parent.Nodes.GetEnumerator())
					{
						while (enumerator.MoveNext())
						{
							XmlElement xmlElement = enumerator.Current;
							if (!FormDesignSetting.IsContainer(xmlElement.Type.ToString()))
							{
								flag = false;
								break;
							}
						}
						return flag;
					}
					break;
				}
				default:
					return flag;
				}
				if (childType == ComponentType.Group)
				{
					if (parent.Nodes.Count <= 0)
					{
						return flag;
					}
					using (IEnumerator<XmlElement> enumerator2 = parent.Nodes.GetEnumerator())
					{
						while (enumerator2.MoveNext())
						{
							XmlElement xmlElement2 = enumerator2.Current;
							if (xmlElement2.Type != ComponentType.Group)
							{
								flag = false;
								break;
							}
						}
						return flag;
					}
				}
				if (parent.Nodes.Count > 0)
				{
					foreach (XmlElement xmlElement3 in parent.Nodes)
					{
						if (xmlElement3.Type == ComponentType.Group)
						{
							flag = false;
							break;
						}
					}
				}
			}
			return flag;
		}

		// Token: 0x06000940 RID: 2368 RVA: 0x0002ED20 File Offset: 0x0002CF20
		public static bool AcceptMimes(string parentNodeName, string childNodeName)
		{
			if (parentNodeName == ComponentType.Form.ToString())
			{
				return false;
			}
			if (childNodeName == ComponentType.Phantom.ToString())
			{
				return parentNodeName == ComponentType.Tree.ToString() || parentNodeName == ComponentType.Table.ToString();
			}
			string pNodeName = string.Format("modFD/{0}", parentNodeName);
			string text = string.Format("modFD/{0}", childNodeName);
			XElement nodeInfoList = ComponentFactory._nodeInfoList;
			XElement xelement = (from n in nodeInfoList.Elements("NodeInfo")
				where n.Attribute("mimeType").Value == pNodeName
				select n).FirstOrDefault<XElement>();
			if (xelement == null)
			{
				return false;
			}
			string value = xelement.Attribute("acceptedMimes").Value;
			bool flag = value.Split(new char[] { ';' }).Contains(text);
			if (parentNodeName == ComponentType.Page.ToString())
			{
				return flag && FormDesignSetting.IsContainer(childNodeName);
			}
			return flag;
		}

		// Token: 0x06000941 RID: 2369 RVA: 0x0002EE30 File Offset: 0x0002D030
		internal static XmlElement ConvertTo(XmlElement element, ComponentType type)
		{
			Dictionary<string, string> dictionary = new Dictionary<string, string>();
			foreach (string text in element.Attributes)
			{
				dictionary.Add(text, element.GetAttribute(text));
			}
			XmlElement xmlElement = ComponentFactory.CreateEmptyComponent(element.Key, type, element.Name);
			foreach (KeyValuePair<string, string> keyValuePair in dictionary)
			{
				if (!(keyValuePair.Key == "widget") && xmlElement.GetAttribute(keyValuePair.Key) != null)
				{
					xmlElement.SetAttribute(keyValuePair.Key, keyValuePair.Value);
				}
			}
			return xmlElement;
		}

		// Token: 0x04000347 RID: 839
		private static XElement _nodeInfoList;

		// Token: 0x04000348 RID: 840
		private static XElement _propertyInfoList;

		// Token: 0x04000349 RID: 841
		private static ModFdInfo _modFdInfo;

		// Token: 0x0400034A RID: 842
		private static XElement _core_br;
	}
}
