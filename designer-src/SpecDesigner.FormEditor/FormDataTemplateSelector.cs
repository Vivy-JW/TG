using System;
using System.Windows;
using System.Windows.Controls;
using SpecDesignerCommon.Helpers;
using SpecDesignerCommon.ViewModel;

namespace SpecDesigner.FormEditor
{
	// Token: 0x02000015 RID: 21
	public class FormDataTemplateSelector : DataTemplateSelector
	{
		// Token: 0x06000081 RID: 129 RVA: 0x00004DEC File Offset: 0x00002FEC
		public override DataTemplate SelectTemplate(object item, DependencyObject container)
		{
			XmlElement xmlElement = item as XmlElement;
			if (xmlElement == null)
			{
				return null;
			}
			switch (FormSpecModel.GetNodeType(xmlElement))
			{
			case SpecNodeType.PROGREL:
				return this.ProgRelTemplate;
			case SpecNodeType.REFERENCE:
				return this.ReferenceTemplate;
			case SpecNodeType.MULTILANG:
				return this.MultiLangTemplate;
			}
			switch (xmlElement.Type)
			{
			case ComponentType.Folder:
				return this.FolderTemplate;
			case ComponentType.Grid:
				return this.GridTemplate;
			case ComponentType.Group:
				return this.GroupTemplate;
			case ComponentType.HBox:
				return this.HBoxTemplate;
			case ComponentType.Page:
				return this.PageTemplate;
			case ComponentType.RadioGroup:
				return this.RadioGroupTemplate;
			case ComponentType.ScrollGrid:
				return this.ScrollGridTemplate;
			case ComponentType.Table:
				return this.TableTemplate;
			case ComponentType.Tree:
				return this.TreeTemplate;
			case ComponentType.VBox:
				return this.VBoxTemplate;
			case ComponentType.Label:
				return this.LabelTemplate;
			case ComponentType.Edit:
			case ComponentType.TextEdit:
				return this.EditTemplate;
			case ComponentType.ProgressBar:
				return this.ProgressBarTemplate;
			case ComponentType.ComboBox:
				return this.ComboBoxTemplate;
			case ComponentType.Button:
				return this.ButtonTemplate;
			case ComponentType.ButtonEdit:
				return this.ButtonEditTemplate;
			case ComponentType.DateEdit:
				return this.DateEditTemplate;
			case ComponentType.RadioGroupItem:
				return this.RadioGroupItemTemplate;
			case ComponentType.Canvas:
				return this.CanvasTemplate;
			case ComponentType.CheckBox:
				return this.CheckBoxTemplate;
			case ComponentType.FFLabel:
				return this.FFlabelTemplate;
			case ComponentType.FFImage:
				return this.FFImageTemplate;
			case ComponentType.Image:
				return this.ImageTemplate;
			case ComponentType.Slider:
				return this.SliderTemplate;
			case ComponentType.SpinEdit:
				return this.SpinEditTemplate;
			case ComponentType.TimeEdit:
				return this.TimeEditTemplate;
			case ComponentType.WebComponent:
				return this.WebComponentTemplate;
			case ComponentType.HLine:
				return this.HLineTemplate;
			case ComponentType.Phantom:
				return this.PhantomTemplate;
			case ComponentType.DateTimeEdit:
				return this.DateTimeEditTemplate;
			}
			return null;
		}

		// Token: 0x17000009 RID: 9
		// (get) Token: 0x06000082 RID: 130 RVA: 0x00004FA6 File Offset: 0x000031A6
		// (set) Token: 0x06000083 RID: 131 RVA: 0x00004FAE File Offset: 0x000031AE
		public DataTemplate VBoxTemplate { get; set; }

		// Token: 0x1700000A RID: 10
		// (get) Token: 0x06000084 RID: 132 RVA: 0x00004FB7 File Offset: 0x000031B7
		// (set) Token: 0x06000085 RID: 133 RVA: 0x00004FBF File Offset: 0x000031BF
		public DataTemplate HBoxTemplate { get; set; }

		// Token: 0x1700000B RID: 11
		// (get) Token: 0x06000086 RID: 134 RVA: 0x00004FC8 File Offset: 0x000031C8
		// (set) Token: 0x06000087 RID: 135 RVA: 0x00004FD0 File Offset: 0x000031D0
		public DataTemplate GridTemplate { get; set; }

		// Token: 0x1700000C RID: 12
		// (get) Token: 0x06000088 RID: 136 RVA: 0x00004FD9 File Offset: 0x000031D9
		// (set) Token: 0x06000089 RID: 137 RVA: 0x00004FE1 File Offset: 0x000031E1
		public DataTemplate GroupTemplate { get; set; }

		// Token: 0x1700000D RID: 13
		// (get) Token: 0x0600008A RID: 138 RVA: 0x00004FEA File Offset: 0x000031EA
		// (set) Token: 0x0600008B RID: 139 RVA: 0x00004FF2 File Offset: 0x000031F2
		public DataTemplate FolderTemplate { get; set; }

		// Token: 0x1700000E RID: 14
		// (get) Token: 0x0600008C RID: 140 RVA: 0x00004FFB File Offset: 0x000031FB
		// (set) Token: 0x0600008D RID: 141 RVA: 0x00005003 File Offset: 0x00003203
		public DataTemplate ScrollGridTemplate { get; set; }

		// Token: 0x1700000F RID: 15
		// (get) Token: 0x0600008E RID: 142 RVA: 0x0000500C File Offset: 0x0000320C
		// (set) Token: 0x0600008F RID: 143 RVA: 0x00005014 File Offset: 0x00003214
		public DataTemplate TableTemplate { get; set; }

		// Token: 0x17000010 RID: 16
		// (get) Token: 0x06000090 RID: 144 RVA: 0x0000501D File Offset: 0x0000321D
		// (set) Token: 0x06000091 RID: 145 RVA: 0x00005025 File Offset: 0x00003225
		public DataTemplate TreeTemplate { get; set; }

		// Token: 0x17000011 RID: 17
		// (get) Token: 0x06000092 RID: 146 RVA: 0x0000502E File Offset: 0x0000322E
		// (set) Token: 0x06000093 RID: 147 RVA: 0x00005036 File Offset: 0x00003236
		public DataTemplate RadioGroupTemplate { get; set; }

		// Token: 0x17000012 RID: 18
		// (get) Token: 0x06000094 RID: 148 RVA: 0x0000503F File Offset: 0x0000323F
		// (set) Token: 0x06000095 RID: 149 RVA: 0x00005047 File Offset: 0x00003247
		public DataTemplate RadioGroupItemTemplate { get; set; }

		// Token: 0x17000013 RID: 19
		// (get) Token: 0x06000096 RID: 150 RVA: 0x00005050 File Offset: 0x00003250
		// (set) Token: 0x06000097 RID: 151 RVA: 0x00005058 File Offset: 0x00003258
		public DataTemplate CanvasTemplate { get; set; }

		// Token: 0x17000014 RID: 20
		// (get) Token: 0x06000098 RID: 152 RVA: 0x00005061 File Offset: 0x00003261
		// (set) Token: 0x06000099 RID: 153 RVA: 0x00005069 File Offset: 0x00003269
		public DataTemplate SpinEditTemplate { get; set; }

		// Token: 0x17000015 RID: 21
		// (get) Token: 0x0600009A RID: 154 RVA: 0x00005072 File Offset: 0x00003272
		// (set) Token: 0x0600009B RID: 155 RVA: 0x0000507A File Offset: 0x0000327A
		public DataTemplate TimeEditTemplate { get; set; }

		// Token: 0x17000016 RID: 22
		// (get) Token: 0x0600009C RID: 156 RVA: 0x00005083 File Offset: 0x00003283
		// (set) Token: 0x0600009D RID: 157 RVA: 0x0000508B File Offset: 0x0000328B
		public DataTemplate SliderTemplate { get; set; }

		// Token: 0x17000017 RID: 23
		// (get) Token: 0x0600009E RID: 158 RVA: 0x00005094 File Offset: 0x00003294
		// (set) Token: 0x0600009F RID: 159 RVA: 0x0000509C File Offset: 0x0000329C
		public DataTemplate HLineTemplate { get; set; }

		// Token: 0x17000018 RID: 24
		// (get) Token: 0x060000A0 RID: 160 RVA: 0x000050A5 File Offset: 0x000032A5
		// (set) Token: 0x060000A1 RID: 161 RVA: 0x000050AD File Offset: 0x000032AD
		public DataTemplate ProgressBarTemplate { get; set; }

		// Token: 0x17000019 RID: 25
		// (get) Token: 0x060000A2 RID: 162 RVA: 0x000050B6 File Offset: 0x000032B6
		// (set) Token: 0x060000A3 RID: 163 RVA: 0x000050BE File Offset: 0x000032BE
		public DataTemplate DateEditTemplate { get; set; }

		// Token: 0x1700001A RID: 26
		// (get) Token: 0x060000A4 RID: 164 RVA: 0x000050C7 File Offset: 0x000032C7
		// (set) Token: 0x060000A5 RID: 165 RVA: 0x000050CF File Offset: 0x000032CF
		public DataTemplate ButtonEditTemplate { get; set; }

		// Token: 0x1700001B RID: 27
		// (get) Token: 0x060000A6 RID: 166 RVA: 0x000050D8 File Offset: 0x000032D8
		// (set) Token: 0x060000A7 RID: 167 RVA: 0x000050E0 File Offset: 0x000032E0
		public DataTemplate CheckBoxTemplate { get; set; }

		// Token: 0x1700001C RID: 28
		// (get) Token: 0x060000A8 RID: 168 RVA: 0x000050E9 File Offset: 0x000032E9
		// (set) Token: 0x060000A9 RID: 169 RVA: 0x000050F1 File Offset: 0x000032F1
		public DataTemplate ButtonTemplate { get; set; }

		// Token: 0x1700001D RID: 29
		// (get) Token: 0x060000AA RID: 170 RVA: 0x000050FA File Offset: 0x000032FA
		// (set) Token: 0x060000AB RID: 171 RVA: 0x00005102 File Offset: 0x00003302
		public DataTemplate ImageTemplate { get; set; }

		// Token: 0x1700001E RID: 30
		// (get) Token: 0x060000AC RID: 172 RVA: 0x0000510B File Offset: 0x0000330B
		// (set) Token: 0x060000AD RID: 173 RVA: 0x00005113 File Offset: 0x00003313
		public DataTemplate FFImageTemplate { get; set; }

		// Token: 0x1700001F RID: 31
		// (get) Token: 0x060000AE RID: 174 RVA: 0x0000511C File Offset: 0x0000331C
		// (set) Token: 0x060000AF RID: 175 RVA: 0x00005124 File Offset: 0x00003324
		public DataTemplate ComboBoxTemplate { get; set; }

		// Token: 0x17000020 RID: 32
		// (get) Token: 0x060000B0 RID: 176 RVA: 0x0000512D File Offset: 0x0000332D
		// (set) Token: 0x060000B1 RID: 177 RVA: 0x00005135 File Offset: 0x00003335
		public DataTemplate WebComponentTemplate { get; set; }

		// Token: 0x17000021 RID: 33
		// (get) Token: 0x060000B2 RID: 178 RVA: 0x0000513E File Offset: 0x0000333E
		// (set) Token: 0x060000B3 RID: 179 RVA: 0x00005146 File Offset: 0x00003346
		public DataTemplate PageTemplate { get; set; }

		// Token: 0x17000022 RID: 34
		// (get) Token: 0x060000B4 RID: 180 RVA: 0x0000514F File Offset: 0x0000334F
		// (set) Token: 0x060000B5 RID: 181 RVA: 0x00005157 File Offset: 0x00003357
		public DataTemplate PhantomTemplate { get; set; }

		// Token: 0x17000023 RID: 35
		// (get) Token: 0x060000B6 RID: 182 RVA: 0x00005160 File Offset: 0x00003360
		// (set) Token: 0x060000B7 RID: 183 RVA: 0x00005168 File Offset: 0x00003368
		public DataTemplate EditTemplate { get; set; }

		// Token: 0x17000024 RID: 36
		// (get) Token: 0x060000B8 RID: 184 RVA: 0x00005171 File Offset: 0x00003371
		// (set) Token: 0x060000B9 RID: 185 RVA: 0x00005179 File Offset: 0x00003379
		public DataTemplate FFlabelTemplate { get; set; }

		// Token: 0x17000025 RID: 37
		// (get) Token: 0x060000BA RID: 186 RVA: 0x00005182 File Offset: 0x00003382
		// (set) Token: 0x060000BB RID: 187 RVA: 0x0000518A File Offset: 0x0000338A
		public DataTemplate LabelTemplate { get; set; }

		// Token: 0x17000026 RID: 38
		// (get) Token: 0x060000BC RID: 188 RVA: 0x00005193 File Offset: 0x00003393
		// (set) Token: 0x060000BD RID: 189 RVA: 0x0000519B File Offset: 0x0000339B
		public DataTemplate DateTimeEditTemplate { get; set; }

		// Token: 0x17000027 RID: 39
		// (get) Token: 0x060000BE RID: 190 RVA: 0x000051A4 File Offset: 0x000033A4
		// (set) Token: 0x060000BF RID: 191 RVA: 0x000051AC File Offset: 0x000033AC
		public DataTemplate MultiLangTemplate { get; set; }

		// Token: 0x17000028 RID: 40
		// (get) Token: 0x060000C0 RID: 192 RVA: 0x000051B5 File Offset: 0x000033B5
		// (set) Token: 0x060000C1 RID: 193 RVA: 0x000051BD File Offset: 0x000033BD
		public DataTemplate ProgRelTemplate { get; set; }

		// Token: 0x17000029 RID: 41
		// (get) Token: 0x060000C2 RID: 194 RVA: 0x000051C6 File Offset: 0x000033C6
		// (set) Token: 0x060000C3 RID: 195 RVA: 0x000051CE File Offset: 0x000033CE
		public DataTemplate ReferenceTemplate { get; set; }
	}
}
